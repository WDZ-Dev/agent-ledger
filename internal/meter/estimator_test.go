package meter

import "testing"

func TestEstimatorCountTokens(t *testing.T) {
	e := NewEstimator()

	tests := []struct {
		name  string
		model string
		text  string
	}{
		{"gpt-4o short", "gpt-4o", "Hello, world!"},
		{"gpt-4o-mini", "gpt-4o-mini", "This is a test of token counting."},
		{"gpt-4", "gpt-4", "Testing cl100k_base encoding."},
		{"claude approximation", "claude-sonnet-4-20250514", "Testing claude estimation."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := e.CountTokens(tt.model, tt.text)
			if count == 0 {
				t.Errorf("expected > 0 tokens for model %q, text %q", tt.model, tt.text)
			}
		})
	}
}

func TestEstimatorUnknownModel(t *testing.T) {
	e := NewEstimator()
	count := e.CountTokens("totally-unknown-model", "Hello")
	if count != 0 {
		t.Errorf("expected 0 for unknown model, got %d", count)
	}
}

func TestEstimatorEmpty(t *testing.T) {
	e := NewEstimator()
	count := e.CountTokens("gpt-4o", "")
	if count != 0 {
		t.Errorf("expected 0 for empty text, got %d", count)
	}
}

func TestEstimatorCaching(t *testing.T) {
	e := NewEstimator()

	// First call creates encoder, second uses cache.
	c1 := e.CountTokens("gpt-4o", "Hello")
	c2 := e.CountTokens("gpt-4o", "Hello")
	if c1 != c2 {
		t.Errorf("results differ: %d vs %d", c1, c2)
	}
}

func TestMeterEstimateTokens(t *testing.T) {
	m := New()
	count := m.EstimateTokens("gpt-4o", "Hello, how are you today?")
	if count == 0 {
		t.Error("expected > 0 from meter's EstimateTokens")
	}
}

func TestEncodingForModel(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"gpt-4o", "o200k_base"},
		{"gpt-4o-mini-2024-07-18", "o200k_base"},
		{"o1-preview", "o200k_base"},
		{"o3-mini", "o200k_base"},
		{"gpt-4-turbo", "cl100k_base"},
		{"gpt-3.5-turbo", "cl100k_base"},
		{"claude-sonnet-4-20250514", "cl100k_base"},
		{"unknown-model", ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := encodingForModel(tt.model)
			if got != tt.want {
				t.Errorf("encodingForModel(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}
