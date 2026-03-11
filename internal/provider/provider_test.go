package provider

import (
	"net/http"
	"testing"
)

func TestHashAPIKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty key", ""},
		{"short key", "sk-abc"},
		{"normal key", "sk-proj-abcdefghijklmnop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HashAPIKey(tt.key)
			if tt.key == "" && h != "" {
				t.Errorf("expected empty hash for empty key, got %q", h)
			}
			if tt.key != "" && h == "" {
				t.Errorf("expected non-empty hash for key %q", tt.key)
			}
			// Same input should produce same output.
			if h2 := HashAPIKey(tt.key); h != h2 {
				t.Errorf("hash not deterministic: %q != %q", h, h2)
			}
		})
	}

	// Different keys should produce different hashes.
	h1 := HashAPIKey("sk-proj-aaaa1111bbbb2222")
	h2 := HashAPIKey("sk-proj-cccc3333dddd4444")
	if h1 == h2 {
		t.Errorf("different keys produced same hash: %q", h1)
	}
}

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   string
	}{
		{
			"openai bearer",
			http.Header{"Authorization": {"Bearer sk-proj-abc123"}},
			"sk-proj-abc123",
		},
		{
			"anthropic x-api-key",
			http.Header{"X-Api-Key": {"sk-ant-api03-xyz"}},
			"sk-ant-api03-xyz",
		},
		{
			"no key",
			http.Header{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: tt.header}
			got := ExtractAPIKey(r)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractAgentHeaders(t *testing.T) {
	r := &http.Request{
		Header: http.Header{
			"X-Agent-Id":      {"code-reviewer"},
			"X-Agent-Session": {"sess_abc123"},
			"X-Agent-User":    {"user@test.com"},
			"X-Agent-Task":    {"Review PR #456"},
		},
		URL: mustParseURL("http://localhost/v1/chat/completions"),
	}

	agentID, sessionID, userID, task := ExtractAgentHeaders(r)
	if agentID != "code-reviewer" {
		t.Errorf("agentID = %q, want %q", agentID, "code-reviewer")
	}
	if sessionID != "sess_abc123" {
		t.Errorf("sessionID = %q, want %q", sessionID, "sess_abc123")
	}
	if userID != "user@test.com" {
		t.Errorf("userID = %q, want %q", userID, "user@test.com")
	}
	if task != "Review PR #456" {
		t.Errorf("task = %q, want %q", task, "Review PR #456")
	}
}

func TestExtractAgentHeaders_QueryParamFallback(t *testing.T) {
	r := &http.Request{
		Header: http.Header{},
		URL:    mustParseURL("http://localhost/v1/chat/completions?_agent_id=qa-bot&_agent_session=sess_qp&_agent_user=ci@test.com&_agent_task=Run+tests"),
	}

	agentID, sessionID, userID, task := ExtractAgentHeaders(r)
	if agentID != "qa-bot" {
		t.Errorf("agentID = %q, want %q", agentID, "qa-bot")
	}
	if sessionID != "sess_qp" {
		t.Errorf("sessionID = %q, want %q", sessionID, "sess_qp")
	}
	if userID != "ci@test.com" {
		t.Errorf("userID = %q, want %q", userID, "ci@test.com")
	}
	if task != "Run tests" {
		t.Errorf("task = %q, want %q", task, "Run tests")
	}
}

func TestExtractAgentHeaders_HeadersOverrideQueryParams(t *testing.T) {
	r := &http.Request{
		Header: http.Header{
			"X-Agent-Id":      {"from-header"},
			"X-Agent-Session": {"sess_header"},
		},
		URL: mustParseURL("http://localhost/v1/chat/completions?_agent_id=from-query&_agent_session=sess_query&_agent_user=qp@test.com"),
	}

	agentID, sessionID, userID, _ := ExtractAgentHeaders(r)
	if agentID != "from-header" {
		t.Errorf("agentID = %q, want %q (headers should take priority)", agentID, "from-header")
	}
	if sessionID != "sess_header" {
		t.Errorf("sessionID = %q, want %q (headers should take priority)", sessionID, "sess_header")
	}
	// userID only in query params — should fall back.
	if userID != "qp@test.com" {
		t.Errorf("userID = %q, want %q (should fall back to query param)", userID, "qp@test.com")
	}
}

func TestStripAgentHeaders(t *testing.T) {
	r := &http.Request{
		Header: http.Header{
			"X-Agent-Id":          {"code-reviewer"},
			"X-Agent-Session":     {"sess_abc123"},
			"X-Agent-Session-End": {"true"},
			"Authorization":       {"Bearer sk-proj-abc"},
		},
	}

	StripAgentHeaders(r)

	if r.Header.Get("X-Agent-Id") != "" {
		t.Error("X-Agent-Id should be stripped")
	}
	if r.Header.Get("X-Agent-Session") != "" {
		t.Error("X-Agent-Session should be stripped")
	}
	if r.Header.Get("Authorization") != "Bearer sk-proj-abc" {
		t.Error("Authorization should NOT be stripped")
	}
}
