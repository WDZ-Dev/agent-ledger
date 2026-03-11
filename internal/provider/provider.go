package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// Provider defines how to parse requests and responses for a specific LLM API.
type Provider interface {
	// Name returns the provider identifier (e.g., "openai", "anthropic").
	Name() string

	// Match returns true if this provider should handle the given request.
	Match(r *http.Request) bool

	// ParseRequest extracts metadata from the request body.
	ParseRequest(body []byte) (*RequestMeta, error)

	// ParseResponse extracts token usage from a non-streaming response body.
	ParseResponse(body []byte) (*ResponseMeta, error)

	// ParseStreamChunk extracts token usage from a single SSE event.
	ParseStreamChunk(eventType string, data []byte) (*StreamChunkMeta, error)

	// UpstreamURL returns the base URL of the upstream API.
	UpstreamURL() string
}

// RequestMeta holds metadata extracted from a request body.
type RequestMeta struct {
	Model     string
	MaxTokens int
	Stream    bool
}

// ResponseMeta holds token usage from a complete (non-streaming) response.
type ResponseMeta struct {
	Model        string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// StreamChunkMeta holds token usage from a single SSE chunk.
type StreamChunkMeta struct {
	Model        string
	InputTokens  int
	OutputTokens int
	Done         bool
}

// ExtractAPIKey reads the API key from request headers.
// OpenAI uses Authorization: Bearer, Anthropic uses x-api-key.
func ExtractAPIKey(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if key := r.Header.Get("x-api-key"); key != "" {
		return key
	}
	return ""
}

// HashAPIKey creates a non-reversible fingerprint from an API key.
// Uses the prefix and suffix of the key to allow identification without
// storing the full secret.
func HashAPIKey(key string) string {
	if key == "" {
		return ""
	}
	var fingerprint string
	if len(key) >= 12 {
		fingerprint = key[:8] + key[len(key)-4:]
	} else {
		fingerprint = key
	}
	h := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(h[:12])
}

// ExtractAgentHeaders reads agent identification headers and returns them.
// These headers are stripped before forwarding to upstream.
func ExtractAgentHeaders(r *http.Request) (agentID, sessionID, userID, task string) {
	agentID = r.Header.Get("X-Agent-Id")
	sessionID = r.Header.Get("X-Agent-Session")
	userID = r.Header.Get("X-Agent-User")
	task = r.Header.Get("X-Agent-Task")
	return
}

// StripAgentHeaders removes agent identification headers so they are not
// forwarded to the upstream provider.
func StripAgentHeaders(r *http.Request) {
	r.Header.Del("X-Agent-Id")
	r.Header.Del("X-Agent-Session")
	r.Header.Del("X-Agent-User")
	r.Header.Del("X-Agent-Task")
	r.Header.Del("X-Agent-Session-End")
}
