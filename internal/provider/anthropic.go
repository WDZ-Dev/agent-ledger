package provider

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Anthropic implements the Provider interface for Anthropic's Messages API.
type Anthropic struct {
	upstream string
}

// NewAnthropic creates a new Anthropic provider with the given upstream URL.
func NewAnthropic(upstream string) *Anthropic {
	return &Anthropic{upstream: upstream}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) Match(r *http.Request) bool {
	p := r.URL.Path
	// Anthropic Messages API; also check for the anthropic-version header
	// to distinguish from OpenAI when paths could overlap.
	if strings.HasPrefix(p, "/v1/messages") {
		return r.Header.Get("anthropic-version") != "" ||
			r.Header.Get("x-api-key") != ""
	}
	return false
}

func (a *Anthropic) UpstreamURL() string { return a.upstream }

// anthropicRequest is the minimal subset of an Anthropic messages request.
type anthropicRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Stream    bool   `json:"stream"`
}

func (a *Anthropic) ParseRequest(body []byte) (*RequestMeta, error) {
	var req anthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &RequestMeta{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}, nil
}

// anthropicResponse is the minimal subset of an Anthropic messages response.
type anthropicResponse struct {
	Model string `json:"model"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *Anthropic) ParseResponse(body []byte) (*ResponseMeta, error) {
	var resp anthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &ResponseMeta{
		Model:        resp.Model,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}, nil
}

// Anthropic streaming uses typed SSE events:
//   event: message_start   → has input_tokens in usage
//   event: message_delta   → has output_tokens in usage
//   event: message_stop    → done

type anthropicStreamEvent struct {
	Type    string `json:"type"`
	Message *struct {
		Model string `json:"model"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage,omitempty"`
	} `json:"message,omitempty"`
	Usage *struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

func (a *Anthropic) ParseStreamChunk(eventType string, data []byte) (*StreamChunkMeta, error) {
	var event anthropicStreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}

	meta := &StreamChunkMeta{}

	switch eventType {
	case "message_start":
		if event.Message != nil {
			meta.Model = event.Message.Model
			if event.Message.Usage != nil {
				meta.InputTokens = event.Message.Usage.InputTokens
			}
		}
	case "message_delta":
		if event.Usage != nil {
			meta.OutputTokens = event.Usage.OutputTokens
		}
	case "message_stop":
		meta.Done = true
	}

	return meta, nil
}
