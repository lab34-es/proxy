package proxy

import (
	"encoding/json"
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

func setupGuardrailForwarder(t *testing.T) (*Forwarder, *store.GuardrailStore, *store.GuardrailEventStore, string, string) {
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
	// Set cache TTL to 0 to force fresh loads in tests.
	f.cacheTTL = 0
	return f, gs, ges, key.ID, provider.ID
}

func TestGuardrail_RejectMode(t *testing.T) {
	f, gs, ges, keyID, providerID := setupGuardrailForwarder(t)

	// Create a reject guardrail.
	_, err := gs.Create(`\bpassword\b`, "reject", "")
	require.NoError(t, err)

	// Mock upstream (should NOT be reached).
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream should not be called for rejected requests")
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"my password is secret123"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err = f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.Error(t, err)

	// Verify it's a GuardrailRejectedError.
	var grErr *GuardrailRejectedError
	assert.ErrorAs(t, err, &grErr)
	assert.Contains(t, grErr.Error(), "guardrail")

	// Verify event was recorded.
	result, err := ges.List(store.GuardrailEventQuery{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "reject", result.Records[0].Mode)
	assert.Contains(t, result.Records[0].InputText, "password")
}

func TestGuardrail_ReplaceMode(t *testing.T) {
	f, gs, _, keyID, providerID := setupGuardrailForwarder(t)

	// Create a replace guardrail.
	_, err := gs.Create(`secret\d+`, "replace", "[REDACTED]")
	require.NoError(t, err)

	// Mock upstream that echoes back the received messages.
	var receivedBody map[string]json.RawMessage
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
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

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"the code is secret123 and secret456"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err = f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err)

	// Verify the messages were modified before forwarding.
	var messages []map[string]json.RawMessage
	json.Unmarshal(receivedBody["messages"], &messages)
	require.Len(t, messages, 1)
	assert.Equal(t, "the code is [REDACTED] and [REDACTED]", extractTextContent(messages[0]["content"]))
}

func TestGuardrail_NoMatchPassesThrough(t *testing.T) {
	f, gs, _, keyID, providerID := setupGuardrailForwarder(t)

	// Create a reject guardrail that won't match.
	_, err := gs.Create(`\bpassword\b`, "reject", "")
	require.NoError(t, err)

	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
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

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"hello world"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err = f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err)
	assert.True(t, upstreamCalled)
}

func TestGuardrail_RejectOnSystemMessage(t *testing.T) {
	f, gs, _, keyID, providerID := setupGuardrailForwarder(t)

	_, err := gs.Create(`\bpassword\b`, "reject", "")
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream should not be called")
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	// password in system message, not user message.
	body := `{"model":"gpt-4","messages":[{"role":"system","content":"your password is admin"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err = f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.Error(t, err)
	var grErr *GuardrailRejectedError
	assert.ErrorAs(t, err, &grErr)
}

func TestGuardrail_ReplaceOnAssistantMessage(t *testing.T) {
	f, gs, _, keyID, providerID := setupGuardrailForwarder(t)

	_, err := gs.Create(`confidential`, "replace", "[REDACTED]")
	require.NoError(t, err)

	var receivedBody map[string]json.RawMessage
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
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

	body := `{"model":"gpt-4","messages":[{"role":"assistant","content":"this is confidential data"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err = f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err)

	var messages []map[string]json.RawMessage
	json.Unmarshal(receivedBody["messages"], &messages)
	require.Len(t, messages, 1)
	assert.Equal(t, "this is [REDACTED] data", extractTextContent(messages[0]["content"]))
}

func TestGuardrail_MultipleRules(t *testing.T) {
	f, gs, _, keyID, providerID := setupGuardrailForwarder(t)

	// Replace rule applied first.
	_, err := gs.Create(`secret`, "replace", "***")
	require.NoError(t, err)
	// Reject rule.
	_, err = gs.Create(`\bpassword\b`, "reject", "")
	require.NoError(t, err)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream should not be called")
	}))
	defer upstream.Close()

	provider := &models.Provider{
		ID:      providerID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := &models.APIKey{ID: keyID, ProviderID: providerID}

	// Message contains both "secret" and "password". The reject should fire.
	body := `{"model":"gpt-4","messages":[{"role":"user","content":"my password is secret"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err = f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.Error(t, err)
	var grErr *GuardrailRejectedError
	assert.ErrorAs(t, err, &grErr)
}

func TestGuardrail_NoGuardrails(t *testing.T) {
	f, _, _, keyID, providerID := setupGuardrailForwarder(t)

	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
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

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"anything goes"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err := f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err)
	assert.True(t, upstreamCalled)
}

func TestGuardrail_CacheRefresh(t *testing.T) {
	f, gs, _, keyID, providerID := setupGuardrailForwarder(t)

	// Initially no guardrails.
	messages := []json.RawMessage{json.RawMessage(`{"role":"user","content":"password"}`)}
	result, err := f.applyGuardrails(messages, keyID)
	require.NoError(t, err)
	assert.Len(t, result, 1) // no change

	// Add a guardrail - should be picked up since cacheTTL is 0.
	_, err = gs.Create(`password`, "replace", "***")
	require.NoError(t, err)

	result, err = f.applyGuardrails(messages, keyID)
	require.NoError(t, err)
	var hdr chatMessageHeader
	json.Unmarshal(result[0], &hdr)
	assert.Equal(t, "***", extractTextContent(hdr.Content))
	_ = providerID
}

func TestGuardrail_ToolCallFieldsPreservedDuringReplace(t *testing.T) {
	f, gs, _, keyID, providerID := setupGuardrailForwarder(t)

	// Replace guardrail that matches user message content.
	_, err := gs.Create(`weather`, "replace", "climate")
	require.NoError(t, err)

	var receivedBody map[string]json.RawMessage
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
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

	// Conversation with tool calls; the user message contains "weather" which triggers replace.
	body := `{
		"model": "gpt-4",
		"messages": [
			{"role": "user", "content": "What is the weather?"},
			{"role": "assistant", "content": null, "tool_calls": [{"id": "call_abc123", "type": "function", "function": {"name": "get_weather", "arguments": "{}"}}]},
			{"role": "tool", "tool_call_id": "call_abc123", "content": "{\"temp\": 72}"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	err = f.ForwardChatCompletion(rec, req, provider, apiKey)
	require.NoError(t, err)

	// Parse the upstream body.
	var messages []map[string]json.RawMessage
	json.Unmarshal(receivedBody["messages"], &messages)
	require.Len(t, messages, 3)

	// User message should have been replaced.
	var userContent string
	json.Unmarshal(messages[0]["content"], &userContent)
	assert.Equal(t, "What is the climate?", userContent)

	// Assistant tool_calls must be preserved.
	assert.NotNil(t, messages[1]["tool_calls"], "assistant tool_calls must survive guardrail replace")
	assert.Contains(t, string(messages[1]["tool_calls"]), "call_abc123")

	// Tool message tool_call_id must be preserved.
	assert.NotNil(t, messages[2]["tool_call_id"], "tool_call_id must survive guardrail replace")
	assert.Contains(t, string(messages[2]["tool_call_id"]), "call_abc123")
}

func TestGuardrail_ToolMessageSkippedByGuardrails(t *testing.T) {
	f, gs, _, keyID, _ := setupGuardrailForwarder(t)

	// A guardrail that would match tool output content.
	_, err := gs.Create(`temp`, "reject", "")
	require.NoError(t, err)

	// Tool message contains "temp" but should be skipped by guardrails.
	messages := []json.RawMessage{
		json.RawMessage(`{"role":"user","content":"What is the weather?"}`),
		json.RawMessage(`{"role":"tool","tool_call_id":"call_abc123","content":"{\"temp\": 72}"}`),
	}

	result, err := f.applyGuardrails(messages, keyID)
	require.NoError(t, err)
	assert.Len(t, result, 2, "tool message should not be rejected")
}

func TestGuardrailRejectedError_Message(t *testing.T) {
	err := &GuardrailRejectedError{Pattern: "test"}
	assert.Contains(t, err.Error(), "guardrail")
	assert.Contains(t, err.Error(), "test")
}
