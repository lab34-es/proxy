package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lab34-es/proxy/internal/db"
	"github.com/lab34-es/proxy/internal/models"
	"github.com/lab34-es/proxy/internal/store"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// benchEnv holds all the state for a realistic benchmark environment with
// 100 providers, 100 API keys, and 100 guardrails.
type benchEnv struct {
	forwarder  *Forwarder
	providers  []*models.Provider
	apiKeys    []*models.APIKey
	rawKeys    []string
	keyStore   *store.APIKeyStore
	usageStore *store.UsageStore
}

func setupBenchEnv(b *testing.B) *benchEnv {
	b.Helper()

	database, err := db.Open(":memory:")
	require.NoError(b, err)
	b.Cleanup(func() { database.Close() })

	ps := store.NewProviderStore(database)
	ks := store.NewAPIKeyStore(database)
	us := store.NewUsageStore(database)
	gs := store.NewGuardrailStore(database)
	ges := store.NewGuardrailEventStore(database)

	env := &benchEnv{
		keyStore:   ks,
		usageStore: us,
	}

	// Create 100 providers.
	for i := 0; i < 100; i++ {
		p, err := ps.Create(
			fmt.Sprintf("provider-%03d", i),
			fmt.Sprintf("https://provider-%03d.example.com", i),
			fmt.Sprintf("sk-provider-%03d", i),
		)
		require.NoError(b, err)
		env.providers = append(env.providers, p)
	}

	// Create 100 API keys (one per provider, no rate limit).
	for i := 0; i < 100; i++ {
		k, raw, err := ks.Create(
			fmt.Sprintf("key-%03d", i),
			env.providers[i].ID,
			0, // unlimited rate
		)
		require.NoError(b, err)
		env.apiKeys = append(env.apiKeys, k)
		env.rawKeys = append(env.rawKeys, raw)
	}

	// Create 100 guardrails: mix of reject (30) and replace (70) with
	// realistic regex patterns that won't match normal chat messages.
	rejectPatterns := []string{
		`(?i)\bpassword\s*[:=]\s*\S+`,
		`(?i)\bsecret[-_]?key\b`,
		`\b\d{3}-\d{2}-\d{4}\b`,                 // SSN
		`(?i)\bdrop\s+table\b`,                  // SQL injection
		`(?i)\bexec\s*\(`,                       // code injection
		`(?i)\b(hack|exploit|vulnerability)\b`,  // security terms
		`(?i)\bapi[-_]?key\s*[:=]\s*\S+`,        // API key leak
		`(?i)\btoken\s*[:=]\s*[a-zA-Z0-9]{20,}`, // token leak
		`(?i)\bprivate[-_]?key\b`,               // private key mention
		`(?i)\bcredential\b`,                    // credential mention
	}
	replacePatterns := []string{
		`(?i)\b(damn|hell|crap)\b`,
		`(?i)\bstupid\b`,
		`(?i)\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`, // credit card
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, // email
		`(?i)\b(?:https?://)\S+`,                         // URLs
		`(?i)\bkill\b`,
		`(?i)\b(idiot|moron)\b`,
	}

	for i := 0; i < 100; i++ {
		if i < 30 {
			pattern := rejectPatterns[i%len(rejectPatterns)]
			_, err := gs.Create(fmt.Sprintf("%s_%d", pattern, i), "reject", "")
			require.NoError(b, err)
		} else {
			pattern := replacePatterns[i%len(replacePatterns)]
			_, err := gs.Create(fmt.Sprintf("%s_%d", pattern, i), "replace", "[REDACTED]")
			require.NoError(b, err)
		}
	}

	env.forwarder = NewForwarder(us, gs, ges)
	return env
}

// newMockUpstream creates a fast mock upstream that returns a valid
// OpenAI-compatible chat completion response.
func newMockUpstream() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/v1/models") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"data":   []map[string]string{{"id": "gpt-4", "object": "model"}},
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "chatcmpl-bench",
			"model": "gpt-4",
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "Hello! How can I help you today?"}},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 8,
				"total_tokens":      18,
			},
		})
	}))
}

// newMockStreamingUpstream creates a mock upstream that returns an SSE stream.
func newMockStreamingUpstream() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		fmt.Fprintln(w, `data: {"id":"chatcmpl-bench","choices":[{"delta":{"content":"Hello"}}]}`)
		flusher.Flush()
		fmt.Fprintln(w, `data: {"id":"chatcmpl-bench","choices":[{"delta":{"content":"!"}}]}`)
		flusher.Flush()
		fmt.Fprintln(w, `data: {"id":"chatcmpl-bench","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`)
		flusher.Flush()
		fmt.Fprintln(w, `data: [DONE]`)
		flusher.Flush()
	}))
}

// ── Benchmarks ────────────────────────────────────────────────────────────

// BenchmarkForwardChatCompletion_NonStreaming benchmarks the full non-streaming
// proxy path: body parsing, guardrail evaluation (100 rules), upstream
// round-trip, response copying, and usage recording.
func BenchmarkForwardChatCompletion_NonStreaming(b *testing.B) {
	env := setupBenchEnv(b)
	upstream := newMockUpstream()
	defer upstream.Close()

	// Use key[0] → provider[0], point provider at mock upstream.
	provider := &models.Provider{
		ID:      env.providers[0].ID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := env.apiKeys[0]

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"What is the capital of France?"}]}`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		if err := env.forwarder.ForwardChatCompletion(rec, req, provider, apiKey); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkForwardChatCompletion_Streaming benchmarks the full streaming (SSE)
// proxy path with 100 guardrails active.
func BenchmarkForwardChatCompletion_Streaming(b *testing.B) {
	env := setupBenchEnv(b)
	upstream := newMockStreamingUpstream()
	defer upstream.Close()

	provider := &models.Provider{
		ID:      env.providers[0].ID,
		BaseURL: upstream.URL,
		APIKey:  "sk-upstream",
	}
	apiKey := env.apiKeys[0]

	body := `{"model":"gpt-4","stream":true,"messages":[{"role":"user","content":"Tell me a joke"}]}`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		if err := env.forwarder.ForwardChatCompletion(rec, req, provider, apiKey); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAPIKeyLookup benchmarks SHA-256 hashing + DB lookup of a proxy API
// key among 100 stored keys.
func BenchmarkAPIKeyLookup(b *testing.B) {
	env := setupBenchEnv(b)
	// Use the 50th key (middle of the set) for a realistic lookup.
	rawKey := env.rawKeys[49]

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		k, err := env.keyStore.Lookup(rawKey)
		if err != nil {
			b.Fatal(err)
		}
		if k == nil {
			b.Fatal("expected key, got nil")
		}
	}
}

// BenchmarkGuardrailEvaluation benchmarks applying 100 guardrail regex patterns
// to a chat message (no match scenario -- worst case, all patterns checked).
func BenchmarkGuardrailEvaluation(b *testing.B) {
	env := setupBenchEnv(b)

	messages := []json.RawMessage{
		json.RawMessage(`{"role":"system","content":"You are a helpful assistant."}`),
		json.RawMessage(`{"role":"user","content":"What is the capital of France? Please explain the history."}`),
	}

	// Warm the guardrail cache.
	_, _ = env.forwarder.applyGuardrails(messages, env.apiKeys[0].ID)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := env.forwarder.applyGuardrails(messages, env.apiKeys[0].ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUsageRecording benchmarks writing a single usage record to SQLite
// with 100 providers and 100 keys present.
func BenchmarkUsageRecording(b *testing.B) {
	env := setupBenchEnv(b)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		idx := i % 100
		err := env.usageStore.Record(
			env.apiKeys[idx].ID,
			env.providers[idx].ID,
			"gpt-4",
			10, 5, 15,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAdminListProviders benchmarks listing all 100 providers through the
// admin handler (Echo context + JSON serialization).
func BenchmarkAdminListProviders(b *testing.B) {
	database, err := db.Open(":memory:")
	require.NoError(b, err)
	b.Cleanup(func() { database.Close() })

	ps := store.NewProviderStore(database)
	for i := 0; i < 100; i++ {
		_, err := ps.Create(
			fmt.Sprintf("provider-%03d", i),
			fmt.Sprintf("https://provider-%03d.example.com", i),
			fmt.Sprintf("sk-%03d", i),
		)
		require.NoError(b, err)
	}

	e := echo.New()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/admin/providers", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		rows, err := ps.List()
		if err != nil {
			b.Fatal(err)
		}
		c.JSON(http.StatusOK, rows)
	}
}
