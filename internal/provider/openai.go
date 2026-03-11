package provider

import (
	"encoding/json"
	"net/http"
	"strings"
)

// OpenAI implements the Provider interface for OpenAI-compatible APIs.
type OpenAI struct {
	upstream string
}

// NewOpenAI creates a new OpenAI provider with the given upstream URL.
func NewOpenAI(upstream string) *OpenAI {
	return &OpenAI{upstream: upstream}
}

func (o *OpenAI) Name() string { return "openai" }

func (o *OpenAI) Match(r *http.Request) bool {
	p := r.URL.Path
	return strings.HasPrefix(p, "/v1/chat/") ||
		strings.HasPrefix(p, "/v1/completions") ||
		strings.HasPrefix(p, "/v1/embeddings") ||
		strings.HasPrefix(p, "/v1/models")
}

func (o *OpenAI) UpstreamURL() string { return o.upstream }

// openaiRequest is the minimal subset of an OpenAI chat completion request.
type openaiRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Stream    bool   `json:"stream"`
}

func (o *OpenAI) ParseRequest(body []byte) (*RequestMeta, error) {
	var req openaiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &RequestMeta{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}, nil
}

// openaiResponse is the minimal subset of an OpenAI chat completion response.
type openaiResponse struct {
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (o *OpenAI) ParseResponse(body []byte) (*ResponseMeta, error) {
	var resp openaiResponse
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

// openaiStreamChunk is the minimal subset of an OpenAI streaming chunk.
type openaiStreamChunk struct {
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

func (o *OpenAI) ParseStreamChunk(_ string, data []byte) (*StreamChunkMeta, error) {
	var chunk openaiStreamChunk
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
