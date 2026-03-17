package provider

import (
	"net/http"
	"os"
	"testing"
)

func TestGeminiMatch(t *testing.T) {
	g := NewGemini("")

	tests := []struct {
		path string
		want bool
	}{
		{"/gemini/v1beta/models/gemini-2.5-pro:generateContent", true},
		{"/gemini/v1beta/models/gemini-2.5-pro:streamGenerateContent", true},
		{"/gemini/v1/models", true},
		{"/v1/chat/completions", false},
		{"/health", false},
	}

	for _, tt := range tests {
		r := &http.Request{URL: mustParseURL(tt.path), Header: http.Header{}}
		if got := g.Match(r); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestGeminiRewritePath(t *testing.T) {
	g := NewGemini("")
	got := g.RewritePath("/gemini/v1beta/models/gemini-2.5-pro:generateContent")
	want := "/v1beta/models/gemini-2.5-pro:generateContent"
	if got != want {
		t.Errorf("RewritePath = %q, want %q", got, want)
	}
}

func TestExtractGeminiModel(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/gemini/v1beta/models/gemini-2.5-pro:generateContent", "gemini-2.5-pro"},
		{"/gemini/v1beta/models/gemini-2.0-flash:streamGenerateContent", "gemini-2.0-flash"},
		{"/gemini/v1/models", ""},
	}

	for _, tt := range tests {
		got := ExtractGeminiModel(tt.path)
		if got != tt.want {
			t.Errorf("ExtractGeminiModel(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestGeminiParseRequest(t *testing.T) {
	g := NewGemini("")

	body := []byte(`{"contents":[{"parts":[{"text":"Hello"}]}],"generationConfig":{"maxOutputTokens":256}}`)
	meta, err := g.ParseRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.MaxTokens != 256 {
		t.Errorf("max_tokens = %d, want 256", meta.MaxTokens)
	}
}

func TestGeminiParseResponse(t *testing.T) {
	g := NewGemini("")

	body, err := os.ReadFile("testdata/gemini_generate_response.json")
	if err != nil {
		t.Fatal(err)
	}

	meta, err := g.ParseResponse(body)
	if err != nil {
		t.Fatal(err)
	}

	if meta.InputTokens != 8 {
		t.Errorf("input_tokens = %d, want 8", meta.InputTokens)
	}
	if meta.OutputTokens != 12 {
		t.Errorf("output_tokens = %d, want 12", meta.OutputTokens)
	}
	if meta.TotalTokens != 20 {
		t.Errorf("total_tokens = %d, want 20", meta.TotalTokens)
	}
	if meta.Model != "gemini-2.5-pro" {
		t.Errorf("model = %q, want %q", meta.Model, "gemini-2.5-pro")
	}
}

func TestGeminiParseStreamChunk(t *testing.T) {
	g := NewGemini("")

	// Regular chunk with text.
	data := []byte(`{"candidates":[{"content":{"parts":[{"text":"Hi"}],"role":"model"}}]}`)
	chunk, err := g.ParseStreamChunk("", data)
	if err != nil {
		t.Fatal(err)
	}
	if chunk.Text != "Hi" {
		t.Errorf("text = %q, want %q", chunk.Text, "Hi")
	}
	if chunk.Done {
		t.Error("should not be done")
	}

	// Final chunk with usage.
	data = []byte(`{"candidates":[{"content":{"parts":[{"text":"!"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`)
	chunk, err = g.ParseStreamChunk("", data)
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
		t.Error("should be done when usage present")
	}
}

func TestGeminiName(t *testing.T) {
	if NewGemini("").Name() != "gemini" {
		t.Error("name mismatch")
	}
}
