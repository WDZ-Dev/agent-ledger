package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WDZ-Dev/agent-ledger/internal/config"
	"github.com/WDZ-Dev/agent-ledger/internal/ledger"
	"github.com/WDZ-Dev/agent-ledger/internal/meter"
	"github.com/WDZ-Dev/agent-ledger/internal/provider"
)

func BenchmarkNonStreamingProxy(b *testing.B) {
	respBody := `{"id":"chatcmpl-abc","model":"gpt-4o-mini","object":"chat.completion","choices":[{"message":{"content":"Hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, respBody)
	}))
	defer upstream.Close()

	store := &mockStore{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100000, 4, logger)
	defer rec.Close()

	reg := provider.NewRegistry(config.ProvidersConfig{
		OpenAI:    config.ProviderConfig{Upstream: upstream.URL, Enabled: true},
		Anthropic: config.ProviderConfig{Upstream: upstream.URL, Enabled: true},
	})
	m := meter.New()
	p := New(reg, m, rec, nil, nil, nil, nil, nil, nil, nil, logger)

	reqBody := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer sk-proj-test1234abcd")
		w := httptest.NewRecorder()
		p.ServeHTTP(w, req)
	}
}

func BenchmarkStreamingProxy(b *testing.B) {
	ssePayload := "data: {\"id\":\"chatcmpl-abc\",\"model\":\"gpt-4o-mini\",\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n" +
		"data: {\"id\":\"chatcmpl-abc\",\"model\":\"gpt-4o-mini\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n" +
		"data: [DONE]\n\n"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ssePayload)
	}))
	defer upstream.Close()

	store := &mockStore{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100000, 4, logger)
	defer rec.Close()

	reg := provider.NewRegistry(config.ProvidersConfig{
		OpenAI:    config.ProviderConfig{Upstream: upstream.URL, Enabled: true},
		Anthropic: config.ProviderConfig{Upstream: upstream.URL, Enabled: true},
	})
	m := meter.New()
	p := New(reg, m, rec, nil, nil, nil, nil, nil, nil, nil, logger)

	reqBody := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}],"stream":true}`

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer sk-proj-test1234abcd")
		w := httptest.NewRecorder()
		p.ServeHTTP(w, req)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	store := &mockStore{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := ledger.NewRecorder(store, 100, 1, logger)
	defer rec.Close()

	reg := provider.NewRegistry(config.ProvidersConfig{
		OpenAI: config.ProviderConfig{Upstream: upstream.URL, Enabled: true},
	})
	m := meter.New()
	p := New(reg, m, rec, nil, nil, nil, nil, nil, nil, nil, logger)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		p.ServeHTTP(w, req)
	}
}
