package provider

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Cohere implements the Provider interface for Cohere's Chat API (v2).
type Cohere struct {
	upstream string
}

// NewCohere creates a Cohere provider.
func NewCohere(upstream string) *Cohere {
	if upstream == "" {
		upstream = "https://api.cohere.com"
	}
	return &Cohere{upstream: upstream}
}

func (c *Cohere) Name() string        { return "cohere" }
func (c *Cohere) UpstreamURL() string { return c.upstream }
func (c *Cohere) PathPrefix() string  { return "/cohere" }

func (c *Cohere) Match(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/cohere/")
}

// RewritePath strips the /cohere prefix so upstream sees the native path.
func (c *Cohere) RewritePath(path string) string {
	return strings.TrimPrefix(path, "/cohere")
}

// cohereRequest is the minimal subset of a Cohere v2 chat request.
type cohereRequest struct {
	Model     string `json:"model"`
	Stream    bool   `json:"stream"`
	MaxTokens int    `json:"max_tokens"`
}

func (c *Cohere) ParseRequest(body []byte) (*RequestMeta, error) {
	var req cohereRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &RequestMeta{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}, nil
}

// cohereResponse is the minimal subset of a Cohere v2 chat response.
type cohereResponse struct {
	Message *struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message,omitempty"`
	Meta *struct {
		Tokens *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"tokens,omitempty"`
	} `json:"meta,omitempty"`
}

func (c *Cohere) ParseResponse(body []byte) (*ResponseMeta, error) {
	var resp cohereResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	meta := &ResponseMeta{}
	if resp.Meta != nil && resp.Meta.Tokens != nil {
		meta.InputTokens = resp.Meta.Tokens.InputTokens
		meta.OutputTokens = resp.Meta.Tokens.OutputTokens
		meta.TotalTokens = meta.InputTokens + meta.OutputTokens
	}
	return meta, nil
}

// Cohere SSE streams use typed events. The "message-end" event contains usage.
//
// event: content-delta
// data: {"type":"content-delta","delta":{"message":{"content":{"text":"Hi"}}}}
//
// event: message-end
// data: {"type":"message-end","delta":{"finish_reason":"COMPLETE","usage":{"billed_units":{"input_tokens":10,"output_tokens":5}}}}

type cohereStreamEvent struct {
	Type  string `json:"type"`
	Delta *struct {
		Message *struct {
			Content *struct {
				Text string `json:"text"`
			} `json:"content,omitempty"`
		} `json:"message,omitempty"`
		FinishReason string `json:"finish_reason,omitempty"`
		Usage        *struct {
			BilledUnits *struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"billed_units,omitempty"`
		} `json:"usage,omitempty"`
	} `json:"delta,omitempty"`
}

func (c *Cohere) ParseStreamChunk(eventType string, data []byte) (*StreamChunkMeta, error) {
	var event cohereStreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}

	meta := &StreamChunkMeta{}

	switch {
	case eventType == "content-delta" || event.Type == "content-delta":
		if event.Delta != nil && event.Delta.Message != nil && event.Delta.Message.Content != nil {
			meta.Text = event.Delta.Message.Content.Text
		}
	case eventType == "message-end" || event.Type == "message-end":
		meta.Done = true
		if event.Delta != nil && event.Delta.Usage != nil && event.Delta.Usage.BilledUnits != nil {
			meta.InputTokens = event.Delta.Usage.BilledUnits.InputTokens
			meta.OutputTokens = event.Delta.Usage.BilledUnits.OutputTokens
		}
	}

	return meta, nil
}
