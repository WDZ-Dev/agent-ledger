package provider

import (
	"net/http"
	"testing"
)

func TestOpenAICompatMatch_NoPrefix(t *testing.T) {
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

func TestOpenAICompatMatch_WithPrefix(t *testing.T) {
	g := NewGroq("")

	tests := []struct {
		path string
		want bool
	}{
		{"/groq/v1/chat/completions", true},
		{"/groq/v1/completions", true},
		{"/groq/v1/embeddings", true},
		{"/groq/v1/models", true},
		{"/v1/chat/completions", false},
		{"/groq/health", false},
	}

	for _, tt := range tests {
		r := &http.Request{URL: mustParseURL(tt.path), Header: http.Header{}}
		if got := g.Match(r); got != tt.want {
			t.Errorf("Groq.Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestOpenAICompatRewritePath(t *testing.T) {
	tests := []struct {
		name string
		prov *OpenAICompatible
		path string
		want string
	}{
		{"openai no-op", NewOpenAI(""), "/v1/chat/completions", "/v1/chat/completions"},
		{"groq strip", NewGroq(""), "/groq/v1/chat/completions", "/v1/chat/completions"},
		{"mistral strip", NewMistral(""), "/mistral/v1/chat/completions", "/v1/chat/completions"},
		{"deepseek strip", NewDeepSeek(""), "/deepseek/v1/chat/completions", "/v1/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.prov.RewritePath(tt.path)
			if got != tt.want {
				t.Errorf("RewritePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestOpenAICompatProviderNames(t *testing.T) {
	if NewOpenAI("").Name() != "openai" {
		t.Error("OpenAI name mismatch")
	}
	if NewGroq("").Name() != "groq" {
		t.Error("Groq name mismatch")
	}
	if NewMistral("").Name() != "mistral" {
		t.Error("Mistral name mismatch")
	}
	if NewDeepSeek("").Name() != "deepseek" {
		t.Error("DeepSeek name mismatch")
	}
}
