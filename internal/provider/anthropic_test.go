package provider

import (
	"net/http"
	"os"
	"testing"
)

func TestAnthropicMatch(t *testing.T) {
	a := NewAnthropic("https://api.anthropic.com")

	tests := []struct {
		path    string
		headers http.Header
		want    bool
	}{
		{"/v1/messages", http.Header{"Anthropic-Version": {"2023-06-01"}}, true},
		{"/v1/messages", http.Header{"X-Api-Key": {"sk-ant-abc"}}, true},
		{"/v1/messages", http.Header{}, false},
		{"/v1/messages/batch", http.Header{"Anthropic-Version": {"2023-06-01"}}, true},
		{"/v1/chat/completions", http.Header{"Anthropic-Version": {"2023-06-01"}}, false},
	}

	for _, tt := range tests {
		r := &http.Request{URL: mustParseURL(tt.path), Header: tt.headers}
		if got := a.Match(r); got != tt.want {
			t.Errorf("Match(%q, headers=%v) = %v, want %v", tt.path, tt.headers, got, tt.want)
		}
	}
}

func TestAnthropicParseRequest(t *testing.T) {
	a := NewAnthropic("https://api.anthropic.com")

	body := []byte(`{"model":"claude-sonnet-4-20250514","messages":[{"role":"user","content":"hi"}],"max_tokens":1024,"stream":true}`)
	meta, err := a.ParseRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q", meta.Model)
	}
	if meta.MaxTokens != 1024 {
		t.Errorf("max_tokens = %d, want 1024", meta.MaxTokens)
	}
	if !meta.Stream {
		t.Error("stream should be true")
	}
}

func TestAnthropicParseResponse(t *testing.T) {
	a := NewAnthropic("https://api.anthropic.com")

	body, err := os.ReadFile("testdata/anthropic_messages_response.json")
	if err != nil {
		t.Fatal(err)
	}

	meta, err := a.ParseResponse(body)
	if err != nil {
		t.Fatal(err)
	}

	if meta.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q", meta.Model)
	}
	if meta.InputTokens != 15 {
		t.Errorf("input_tokens = %d, want 15", meta.InputTokens)
	}
	if meta.OutputTokens != 10 {
		t.Errorf("output_tokens = %d, want 10", meta.OutputTokens)
	}
	if meta.TotalTokens != 25 {
		t.Errorf("total_tokens = %d, want 25", meta.TotalTokens)
	}
}

func TestAnthropicParseStreamChunks(t *testing.T) {
	a := NewAnthropic("https://api.anthropic.com")

	// message_start with input tokens
	data := []byte(`{"type":"message_start","message":{"id":"msg_abc","model":"claude-sonnet-4-20250514","usage":{"input_tokens":20,"output_tokens":0}}}`)
	chunk, err := a.ParseStreamChunk("message_start", data)
	if err != nil {
		t.Fatal(err)
	}
	if chunk.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q", chunk.Model)
	}
	if chunk.InputTokens != 20 {
		t.Errorf("input_tokens = %d, want 20", chunk.InputTokens)
	}

	// message_delta with output tokens
	data = []byte(`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":15}}`)
	chunk, err = a.ParseStreamChunk("message_delta", data)
	if err != nil {
		t.Fatal(err)
	}
	if chunk.OutputTokens != 15 {
		t.Errorf("output_tokens = %d, want 15", chunk.OutputTokens)
	}

	// message_stop
	data = []byte(`{"type":"message_stop"}`)
	chunk, err = a.ParseStreamChunk("message_stop", data)
	if err != nil {
		t.Fatal(err)
	}
	if !chunk.Done {
		t.Error("should be done on message_stop")
	}
}
