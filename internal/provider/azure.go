package provider

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Azure implements the Provider interface for Azure OpenAI Service.
// Azure uses a different URL scheme: /openai/deployments/{deployment}/chat/completions?api-version=...
// Auth is via api-key header instead of Bearer token.
type Azure struct {
	upstream   string
	pathPrefix string
}

// NewAzure creates an Azure OpenAI provider. The upstream should be the Azure
// resource endpoint (e.g., https://my-resource.openai.azure.com).
// Requests arrive at /azure/openai/deployments/{deployment}/chat/completions.
func NewAzure(upstream string) *Azure {
	return &Azure{
		upstream:   upstream,
		pathPrefix: "/azure",
	}
}

func (a *Azure) Name() string        { return "azure" } //nolint:goconst
func (a *Azure) UpstreamURL() string { return a.upstream }
func (a *Azure) PathPrefix() string  { return a.pathPrefix }

func (a *Azure) Match(r *http.Request) bool {
	p := r.URL.Path
	return strings.HasPrefix(p, a.pathPrefix+"/openai/deployments/")
}

// RewritePath strips the /azure prefix so upstream sees /openai/deployments/...
func (a *Azure) RewritePath(path string) string {
	return strings.TrimPrefix(path, a.pathPrefix)
}

// azureRequest is the minimal subset of an Azure OpenAI request.
type azureRequest struct {
	MaxTokens int  `json:"max_tokens"`
	Stream    bool `json:"stream"`
}

func (a *Azure) ParseRequest(body []byte) (*RequestMeta, error) {
	var req azureRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	// Azure puts the model (deployment) in the URL path, not the body.
	return &RequestMeta{
		Model:     "azure-deployment",
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}, nil
}

// azureResponse matches the Azure OpenAI response (same as OpenAI format).
type azureResponse struct {
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (a *Azure) ParseResponse(body []byte) (*ResponseMeta, error) {
	var resp azureResponse
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

// azureStreamChunk matches the Azure OpenAI streaming chunk (same as OpenAI format).
type azureStreamChunk struct {
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

func (a *Azure) ParseStreamChunk(_ string, data []byte) (*StreamChunkMeta, error) {
	var chunk azureStreamChunk
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
