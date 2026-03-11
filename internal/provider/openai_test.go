package provider

import (
	"net/http"
	"os"
	"testing"
)

func TestOpenAIMatch(t *testing.T) {
	o := NewOpenAI("https://api.openai.com")

	tests := []struct {
		path string
		want bool
	}{
		{"/v1/chat/completions", true},
		{"/v1/completions", true},
		{"/v1/embeddings", true},
		{"/v1/models", true},
		{"/v1/messages", false},
		{"/health", false},
	}

	for _, tt := range tests {
		r := &http.Request{URL: mustParseURL(tt.path), Header: http.Header{}}
		if got := o.Match(r); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestOpenAIParseRequest(t *testing.T) {
	o := NewOpenAI("https://api.openai.com")

	body := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}],"max_tokens":100,"stream":false}`)
	meta, err := o.ParseRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Model != "gpt-4o-mini" {
		t.Errorf("model = %q, want %q", meta.Model, "gpt-4o-mini")
	}
	if meta.MaxTokens != 100 {
		t.Errorf("max_tokens = %d, want %d", meta.MaxTokens, 100)
	}
	if meta.Stream {
		t.Error("stream should be false")
	}
}

func TestOpenAIParseResponse(t *testing.T) {
	o := NewOpenAI("https://api.openai.com")

	body, err := os.ReadFile("testdata/openai_chat_response.json")
	if err != nil {
		t.Fatal(err)
	}

	meta, err := o.ParseResponse(body)
	if err != nil {
		t.Fatal(err)
	}

	if meta.Model != "gpt-4o-mini-2024-07-18" {
		t.Errorf("model = %q", meta.Model)
	}
	if meta.InputTokens != 12 {
		t.Errorf("input_tokens = %d, want 12", meta.InputTokens)
	}
	if meta.OutputTokens != 9 {
		t.Errorf("output_tokens = %d, want 9", meta.OutputTokens)
	}
	if meta.TotalTokens != 21 {
		t.Errorf("total_tokens = %d, want 21", meta.TotalTokens)
	}
}

func TestOpenAIParseStreamChunk(t *testing.T) {
	o := NewOpenAI("https://api.openai.com")

	// Regular chunk without usage.
	data := []byte(`{"id":"chatcmpl-abc","model":"gpt-4o-mini","choices":[{"delta":{"content":"Hi"}}]}`)
	chunk, err := o.ParseStreamChunk("", data)
	if err != nil {
		t.Fatal(err)
	}
	if chunk.Model != "gpt-4o-mini" {
		t.Errorf("model = %q", chunk.Model)
	}
	if chunk.Done {
		t.Error("should not be done")
	}

	// Final chunk with usage (stream_options.include_usage = true).
	data = []byte(`{"id":"chatcmpl-abc","model":"gpt-4o-mini","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	chunk, err = o.ParseStreamChunk("", data)
	if err != nil {
		t.Fatal(err)
	}
	if chunk.InputTokens != 10 {
		t.Errorf("input_tokens = %d, want 10", chunk.InputTokens)
	}
	if chunk.OutputTokens != 5 {
		t.Errorf("output_tokens = %d, want 5", chunk.OutputTokens)
	}
	if !chunk.Done {
		t.Error("should be done when usage is present")
	}
}
