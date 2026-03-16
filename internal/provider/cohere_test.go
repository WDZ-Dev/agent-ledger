package provider

import (
	"net/http"
	"os"
	"testing"
)

func TestCohereMatch(t *testing.T) {
	c := NewCohere("")

	tests := []struct {
		path string
		want bool
	}{
		{"/cohere/v2/chat", true},
		{"/cohere/v1/generate", true},
		{"/v1/chat/completions", false},
		{"/health", false},
	}

	for _, tt := range tests {
		r := &http.Request{URL: mustParseURL(tt.path), Header: http.Header{}}
		if got := c.Match(r); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestCohereRewritePath(t *testing.T) {
	c := NewCohere("")
	got := c.RewritePath("/cohere/v2/chat")
	want := "/v2/chat"
	if got != want {
		t.Errorf("RewritePath = %q, want %q", got, want)
	}
}

func TestCohereParseRequest(t *testing.T) {
	c := NewCohere("")

	body := []byte(`{"model":"command-r-plus","messages":[{"role":"user","content":"hi"}],"stream":true,"max_tokens":512}`)
	meta, err := c.ParseRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Model != "command-r-plus" {
		t.Errorf("model = %q, want %q", meta.Model, "command-r-plus")
	}
	if !meta.Stream {
		t.Error("stream should be true")
	}
	if meta.MaxTokens != 512 {
		t.Errorf("max_tokens = %d, want 512", meta.MaxTokens)
	}
}

func TestCohereParseResponse(t *testing.T) {
	c := NewCohere("")

	body, err := os.ReadFile("testdata/cohere_chat_response.json")
	if err != nil {
		t.Fatal(err)
	}

	meta, err := c.ParseResponse(body)
	if err != nil {
		t.Fatal(err)
	}

	if meta.InputTokens != 14 {
		t.Errorf("input_tokens = %d, want 14", meta.InputTokens)
	}
	if meta.OutputTokens != 8 {
		t.Errorf("output_tokens = %d, want 8", meta.OutputTokens)
	}
	if meta.TotalTokens != 22 {
		t.Errorf("total_tokens = %d, want 22", meta.TotalTokens)
	}
}

func TestCohereParseStreamChunks(t *testing.T) {
	c := NewCohere("")

	// content-delta
	data := []byte(`{"type":"content-delta","delta":{"message":{"content":{"text":"Hello"}}}}`)
	chunk, err := c.ParseStreamChunk("content-delta", data)
	if err != nil {
		t.Fatal(err)
	}
	if chunk.Text != "Hello" {
		t.Errorf("text = %q, want %q", chunk.Text, "Hello")
	}
	if chunk.Done {
		t.Error("should not be done")
	}

	// message-end with usage
	data = []byte(`{"type":"message-end","delta":{"finish_reason":"COMPLETE","usage":{"billed_units":{"input_tokens":10,"output_tokens":5}}}}`)
	chunk, err = c.ParseStreamChunk("message-end", data)
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
		t.Error("should be done on message-end")
	}
}

func TestCohereName(t *testing.T) {
	if NewCohere("").Name() != "cohere" {
		t.Error("name mismatch")
	}
}
