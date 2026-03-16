package provider

import (
	"encoding/json"
	"net/http"
	"strings"
)

// PathRewriter is implemented by providers that use a path prefix for routing.
// After the proxy sets the upstream URL, it calls RewritePath to strip the
// provider prefix so upstream sees the native API path.
type PathRewriter interface {
	RewritePath(path string) string
}

// OpenAICompatible implements the Provider interface for any API that uses the
// OpenAI /v1/chat/completions format (OpenAI, Groq, Mistral, DeepSeek, etc.).
type OpenAICompatible struct {
	name       string
	upstream   string
	pathPrefix string // e.g., "/groq", empty for OpenAI itself
}

// NewOpenAICompatible creates an OpenAI-compatible provider.
func NewOpenAICompatible(name, upstream, pathPrefix string) *OpenAICompatible {
	return &OpenAICompatible{
		name:       name,
		upstream:   upstream,
		pathPrefix: pathPrefix,
	}
}

func (o *OpenAICompatible) Name() string        { return o.name }
func (o *OpenAICompatible) UpstreamURL() string { return o.upstream }
func (o *OpenAICompatible) PathPrefix() string  { return o.pathPrefix }

func (o *OpenAICompatible) Match(r *http.Request) bool {
	p := r.URL.Path
	prefix := o.pathPrefix // e.g., "/groq" or ""
	return strings.HasPrefix(p, prefix+"/v1/chat/") ||
		strings.HasPrefix(p, prefix+"/v1/completions") ||
		strings.HasPrefix(p, prefix+"/v1/embeddings") ||
		strings.HasPrefix(p, prefix+"/v1/models")
}

// RewritePath strips the provider path prefix so upstream sees the native path.
func (o *OpenAICompatible) RewritePath(path string) string {
	if o.pathPrefix == "" {
		return path
	}
	return strings.TrimPrefix(path, o.pathPrefix)
}

// openaiCompatRequest is the minimal subset of an OpenAI chat completion request.
type openaiCompatRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Stream    bool   `json:"stream"`
}

func (o *OpenAICompatible) ParseRequest(body []byte) (*RequestMeta, error) {
	var req openaiCompatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &RequestMeta{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}, nil
}

// openaiCompatResponse is the minimal subset of an OpenAI chat completion response.
type openaiCompatResponse struct {
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (o *OpenAICompatible) ParseResponse(body []byte) (*ResponseMeta, error) {
	var resp openaiCompatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &ResponseMeta{
		Model:        resp.Model,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
	}, nil
}

// openaiCompatStreamChunk is the minimal subset of an OpenAI streaming chunk.
type openaiCompatStreamChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

func (o *OpenAICompatible) ParseStreamChunk(_ string, data []byte) (*StreamChunkMeta, error) {
	var chunk openaiCompatStreamChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil, err
	}

	meta := &StreamChunkMeta{
		Model: chunk.Model,
	}

	if len(chunk.Choices) > 0 {
		meta.Text = chunk.Choices[0].Delta.Content
	}

	if chunk.Usage != nil {
		meta.InputTokens = chunk.Usage.PromptTokens
		meta.OutputTokens = chunk.Usage.CompletionTokens
		meta.Done = true
	}

	return meta, nil
}
