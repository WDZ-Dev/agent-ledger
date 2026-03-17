package provider

import (
	"net/http"
	"testing"
)

func TestAzureMatch(t *testing.T) {
	a := NewAzure("https://my-resource.openai.azure.com")

	tests := []struct {
		path string
		want bool
	}{
		{"/azure/openai/deployments/gpt-4o/chat/completions", true},
		{"/azure/openai/deployments/my-model/completions", true},
		{"/azure/openai/deployments/embed/embeddings", true},
		{"/openai/deployments/gpt-4o/chat/completions", false},
		{"/v1/chat/completions", false},
	}

	for _, tt := range tests {
		r := &http.Request{URL: mustParseURL(tt.path), Header: http.Header{}}
		if got := a.Match(r); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestAzureRewritePath(t *testing.T) {
	a := NewAzure("")
	got := a.RewritePath("/azure/openai/deployments/gpt-4o/chat/completions")
	want := "/openai/deployments/gpt-4o/chat/completions"
	if got != want {
		t.Errorf("RewritePath = %q, want %q", got, want)
	}
}

func TestAzureName(t *testing.T) {
	if NewAzure("").Name() != "azure" {
		t.Error("name mismatch")
	}
}

func TestAzureParseResponse(t *testing.T) {
	a := NewAzure("")
	body := []byte(`{"model":"gpt-4o","usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}`)
	meta, err := a.ParseResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Model != "gpt-4o" {
		t.Errorf("model = %q, want %q", meta.Model, "gpt-4o")
	}
	if meta.InputTokens != 100 {
		t.Errorf("input = %d, want 100", meta.InputTokens)
	}
	if meta.OutputTokens != 50 {
		t.Errorf("output = %d, want 50", meta.OutputTokens)
	}
}

func TestAzureParseRequest(t *testing.T) {
	a := NewAzure("")
	body := []byte(`{"max_tokens":1000,"stream":true}`)
	meta, err := a.ParseRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.MaxTokens != 1000 {
		t.Errorf("max_tokens = %d, want 1000", meta.MaxTokens)
	}
	if !meta.Stream {
		t.Error("stream = false, want true")
	}
}
