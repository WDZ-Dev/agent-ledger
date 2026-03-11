package meter

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// Estimator uses tiktoken to estimate token counts when the API response
// does not include usage data (e.g., streaming without include_usage).
type Estimator struct {
	mu       sync.Mutex
	encoders map[string]*tiktoken.Tiktoken
}

// NewEstimator creates a token estimator backed by tiktoken.
func NewEstimator() *Estimator {
	return &Estimator{
		encoders: make(map[string]*tiktoken.Tiktoken),
	}
}

// CountTokens returns an estimated token count for the given text using
// the encoding appropriate for the model. Returns 0 if the model's
// encoding is unknown.
func (e *Estimator) CountTokens(model, text string) int {
	enc := e.getEncoder(model)
	if enc == nil {
		return 0
	}
	return len(enc.Encode(text, nil, nil))
}

// getEncoder returns a cached tiktoken encoder for the model, creating
// one on first access.
func (e *Estimator) getEncoder(model string) *tiktoken.Tiktoken {
	encoding := encodingForModel(model)
	if encoding == "" {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if enc, ok := e.encoders[encoding]; ok {
		return enc
	}

	enc, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return nil
	}
	e.encoders[encoding] = enc
	return enc
}

// encodingForModel maps model prefixes to tiktoken encoding names.
func encodingForModel(model string) string {
	// cl100k_base: GPT-4, GPT-3.5-turbo, text-embedding-ada-002
	// o200k_base: GPT-4o, o1, o3
	prefixes := []struct {
		prefix   string
		encoding string
	}{
		{"gpt-4.1", "o200k_base"},
		{"gpt-4o", "o200k_base"},
		{"o1", "o200k_base"},
		{"o3", "o200k_base"},
		{"o4", "o200k_base"},
		{"gpt-4", "cl100k_base"},
		{"gpt-3.5", "cl100k_base"},
		{"claude", "cl100k_base"}, // approximation for Anthropic
	}

	for _, p := range prefixes {
		if len(model) >= len(p.prefix) && model[:len(p.prefix)] == p.prefix {
			return p.encoding
		}
	}
	return ""
}
