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
		{"together strip", NewTogether(""), "/together/v1/chat/completions", "/v1/chat/completions"},
		{"fireworks strip", NewFireworks(""), "/fireworks/v1/chat/completions", "/v1/chat/completions"},
		{"perplexity strip", NewPerplexity(""), "/perplexity/v1/chat/completions", "/v1/chat/completions"},
		{"openrouter strip", NewOpenRouter(""), "/openrouter/v1/chat/completions", "/v1/chat/completions"},
		{"xai strip", NewXAI(""), "/xai/v1/chat/completions", "/v1/chat/completions"},
		{"cerebras strip", NewCerebras(""), "/cerebras/v1/chat/completions", "/v1/chat/completions"},
		{"sambanova strip", NewSambaNova(""), "/sambanova/v1/chat/completions", "/v1/chat/completions"},
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
	providers := map[string]*OpenAICompatible{
		"openai":     NewOpenAI(""),
		"groq":       NewGroq(""),
		"mistral":    NewMistral(""),
		"deepseek":   NewDeepSeek(""),
		"together":   NewTogether(""),
		"fireworks":  NewFireworks(""),
		"perplexity": NewPerplexity(""),
		"openrouter": NewOpenRouter(""),
		"xai":        NewXAI(""),
		"cerebras":   NewCerebras(""),
		"sambanova":  NewSambaNova(""),
	}
	for want, p := range providers {
		if p.Name() != want {
			t.Errorf("%s name = %q, want %q", want, p.Name(), want)
		}
	}
}

func TestOpenAICompatNewProviderDefaults(t *testing.T) {
	tests := []struct {
		name     string
		prov     *OpenAICompatible
		upstream string
		prefix   string
	}{
		{"together", NewTogether(""), "https://api.together.xyz", "/together"},
		{"fireworks", NewFireworks(""), "https://api.fireworks.ai/inference", "/fireworks"},
		{"perplexity", NewPerplexity(""), "https://api.perplexity.ai", "/perplexity"},
		{"openrouter", NewOpenRouter(""), "https://openrouter.ai/api", "/openrouter"},
		{"xai", NewXAI(""), "https://api.x.ai", "/xai"},
		{"cerebras", NewCerebras(""), "https://api.cerebras.ai", "/cerebras"},
		{"sambanova", NewSambaNova(""), "https://api.sambanova.ai", "/sambanova"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prov.UpstreamURL() != tt.upstream {
				t.Errorf("upstream = %q, want %q", tt.prov.UpstreamURL(), tt.upstream)
			}
			if tt.prov.PathPrefix() != tt.prefix {
				t.Errorf("prefix = %q, want %q", tt.prov.PathPrefix(), tt.prefix)
			}
		})
	}
}

func TestOpenAICompatMatch_NewProviders(t *testing.T) {
	providers := []*OpenAICompatible{
		NewTogether(""),
		NewFireworks(""),
		NewPerplexity(""),
		NewOpenRouter(""),
		NewXAI(""),
		NewCerebras(""),
		NewSambaNova(""),
	}

	for _, p := range providers {
		prefix := p.PathPrefix()
		r := &http.Request{URL: mustParseURL(prefix + "/v1/chat/completions"), Header: http.Header{}}
		if !p.Match(r) {
			t.Errorf("%s: should match %s/v1/chat/completions", p.Name(), prefix)
		}
		r = &http.Request{URL: mustParseURL("/v1/chat/completions"), Header: http.Header{}}
		if p.Match(r) {
			t.Errorf("%s: should not match /v1/chat/completions", p.Name())
		}
	}
}
