package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lab34-es/proxy/internal/db"
	"github.com/lab34-es/proxy/internal/models"
	"github.com/lab34-es/proxy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupForwarder(t *testing.T) (*Forwarder, *store.UsageStore, string, string) {
	t.Helper()
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	ps := store.NewProviderStore(database)
	ks := store.NewAPIKeyStore(database)
	us := store.NewUsageStore(database)
	gs := store.NewGuardrailStore(database)
	ges := store.NewGuardrailEventStore(database)

	provider, err := ps.Create("test", "https://test.com", "sk-test")
	require.NoError(t, err)
	key, _, err := ks.Create("test-key", provider.ID, 0)
	require.NoError(t, err)

	f := NewForwarder(us, gs, ges)
	return f, us, key.ID, provider.ID
}

func TestForwardChatCompletion_NonStreaming(t *testing.T) {
	f, us, keyID, providerID := setupForwarder(t)

	// Create a mock upstream server.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer sk-upstream", r.Header.Get("Authorization"))

		resp := map[string]interface{}{
			"id":    "chatcmpl-123",
			"model": "gpt-4",
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "Hello!"}},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify usage was recorded.
	result, err := us.Query(store.UsageQuery{APIKeyID: keyID})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 10, result.Records[0].PromptTokens)
	assert.Equal(t, 5, result.Records[0].CompletionTokens)
	assert.Equal(t, 15, result.Records[0].TotalTokens)
}

func TestForwardChatCompletion_Streaming(t *testing.T) {
	f, us, keyID, providerID := setupForwarder(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		// Send SSE chunks.
		fmt.Fprintln(w, `data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"Hi"}}]}`)
		flusher.Flush()
		fmt.Fprintln(w, `data: {"id":"chatcmpl-1","choices":[],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`)
		flusher.Flush()
		fmt.Fprintln(w, `data: [DONE]`)
		flusher.Flush()
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	body := `{"model":"gpt-4","stream":true,"messages":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err)

	// Verify usage was recorded.
	result, err := us.Query(store.UsageQuery{APIKeyID: keyID})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 8, result.Records[0].TotalTokens)
}

func TestForwardChatCompletion_UpstreamError(t *testing.T) {
	f, _, keyID, providerID := setupForwarder(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err) // No error returned, just proxied the error status
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestForwardChatCompletion_InvalidBody(t *testing.T) {
	f, _, keyID, providerID := setupForwarder(t)

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: "https://unused.com",
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	err := f.ForwardChatCompletion(rec, req, provider, apiKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse request")
}

func TestForwardChatCompletion_AcceptHeader(t *testing.T) {
	f, _, keyID, providerID := setupForwarder(t)

	var receivedAccept string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{}})
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	body := `{"model":"gpt-4","messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	err := f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err)
	assert.Equal(t, "application/json", receivedAccept)
}

func TestForwardListModels(t *testing.T) {
	f, _, _, providerID := setupForwarder(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "Bearer sk-upstream", r.Header.Get("Authorization"))

		resp := map[string]interface{}{
			"object": "list",
			"data":   []map[string]string{{"id": "gpt-4", "object": "model"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	err := f.ForwardListModels(rec, req, provider)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "gpt-4")
}

func TestForwardListModels_UpstreamError(t *testing.T) {
	f, _, _, providerID := setupForwarder(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("service unavailable"))
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	err := f.ForwardListModels(rec, req, provider)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestForwardChatCompletion_UnreachableUpstream(t *testing.T) {
	f, _, keyID, providerID := setupForwarder(t)

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: "http://127.0.0.1:1", // unreachable
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	body := `{"model":"gpt-4","messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	rec := httptest.NewRecorder()

	err := f.ForwardChatCompletion(rec, req, provider, apiKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upstream request")
}

func TestForwardListModels_UnreachableUpstream(t *testing.T) {
	f, _, _, providerID := setupForwarder(t)

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: "http://127.0.0.1:1",
		APIKey:  "sk-upstream",
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	err := f.ForwardListModels(rec, req, provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upstream request")
}

func TestForwardChatCompletion_ToolCallFieldsPreserved(t *testing.T) {
	f, _, keyID, providerID := setupForwarder(t)

	// Capture what the upstream actually receives.
	var receivedBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		resp := map[string]interface{}{
			"id":      "chatcmpl-123",
			"model":   "gpt-4",
			"choices": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	// Simulate a conversation with tool calls: user asks something, assistant
	// responds with a tool_call, and then a tool message provides the result.
	body := `{
		"model": "gpt-4",
		"messages": [
			{"role": "user", "content": "What is the weather?"},
			{"role": "assistant", "content": null, "tool_calls": [{"id": "call_abc123", "type": "function", "function": {"name": "get_weather", "arguments": "{\"location\":\"NYC\"}"}}]},
			{"role": "tool", "tool_call_id": "call_abc123", "content": "{\"temp\": 72}"}
		],
		"tools": [{"type": "function", "function": {"name": "get_weather"}}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Parse what the upstream received and verify tool fields are intact.
	var parsed map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(receivedBody, &parsed))

	var messages []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(parsed["messages"], &messages))
	require.Len(t, messages, 3)

	// Message 1 (assistant): must have tool_calls.
	assert.NotNil(t, messages[1]["tool_calls"], "assistant message must preserve tool_calls")
	assert.Contains(t, string(messages[1]["tool_calls"]), "call_abc123")

	// Message 2 (tool): must have tool_call_id.
	assert.NotNil(t, messages[2]["tool_call_id"], "tool message must preserve tool_call_id")
	assert.Contains(t, string(messages[2]["tool_call_id"]), "call_abc123")

	// Verify tools field is also preserved at top level.
	assert.NotNil(t, parsed["tools"], "top-level tools field must be preserved")
}

func TestNewForwarder(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	defer database.Close()

	us := store.NewUsageStore(database)
	gs := store.NewGuardrailStore(database)
	ges := store.NewGuardrailEventStore(database)
	f := NewForwarder(us, gs, ges)
	assert.NotNil(t, f)
	assert.NotNil(t, f.client)
	assert.NotNil(t, f.usage)
	assert.NotNil(t, f.guardrails)
	assert.NotNil(t, f.guardrailEvents)
}
