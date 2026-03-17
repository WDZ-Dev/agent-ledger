package meter

import (
	"math"
	"testing"
)

func TestCalculate(t *testing.T) {
	m := New()

	tests := []struct {
		name         string
		model        string
		inputTokens  int
		outputTokens int
		wantCost     float64
	}{
		{
			"gpt-4o-mini exact",
			"gpt-4o-mini", 1000, 500,
			// (1000/1M * 0.15) + (500/1M * 0.60) = 0.00015 + 0.0003 = 0.00045
			0.00045,
		},
		{
			"gpt-4o-mini versioned (prefix match)",
			"gpt-4o-mini-2024-07-18", 1000, 500,
			0.00045,
		},
		{
			"gpt-4o exact",
			"gpt-4o", 1000, 1000,
			// (1000/1M * 2.50) + (1000/1M * 10.00) = 0.0025 + 0.01 = 0.0125
			0.0125,
		},
		{
			"claude-sonnet-4 prefix match",
			"claude-sonnet-4-20250514", 1000, 1000,
			// (1000/1M * 3.00) + (1000/1M * 15.00) = 0.003 + 0.015 = 0.018
			0.018,
		},
		{
			"unknown model returns 0",
			"unknown-model-xyz", 1000, 1000,
			0,
		},
		{
			"zero tokens",
			"gpt-4o", 0, 0,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.Calculate(tt.model, tt.inputTokens, tt.outputTokens)
			if math.Abs(got-tt.wantCost) > 1e-9 {
				t.Errorf("Calculate(%q, %d, %d) = %f, want %f",
					tt.model, tt.inputTokens, tt.outputTokens, got, tt.wantCost)
			}
		})
	}
}

func TestKnownModel(t *testing.T) {
	m := New()

	if !m.KnownModel("gpt-4o") {
		t.Error("gpt-4o should be known")
	}
	if !m.KnownModel("gpt-4o-mini-2024-07-18") {
		t.Error("gpt-4o-mini-2024-07-18 should match via prefix")
	}
	if m.KnownModel("totally-unknown") {
		t.Error("totally-unknown should not be known")
	}
}

func TestPrefixMatchLongestWins(t *testing.T) {
	m := New()

	// "gpt-4o-mini-2024-07-18" should match "gpt-4o-mini" (len 11),
	// NOT "gpt-4o" (len 6).
	cost := m.Calculate("gpt-4o-mini-2024-07-18", 1_000_000, 0)
	// gpt-4o-mini input: $0.15 per MTok → 1M tokens = $0.15
	if math.Abs(cost-0.15) > 1e-9 {
		t.Errorf("expected $0.15 (gpt-4o-mini pricing), got $%f", cost)
	}
}

func TestNewestModelsKnown(t *testing.T) {
	m := New()

	models := []string{
		// OpenAI — GPT-5 family
		"gpt-5",
		"gpt-5-mini",
		"gpt-5-nano",
		"gpt-5-pro",
		"gpt-5-codex",
		"gpt-5.1",
		"gpt-5.2",
		"gpt-5.2-pro",
		"gpt-5.4",
		"gpt-5.4-pro",
		// OpenAI — reasoning
		"o3-pro",
		"o1-pro",
		"o4-mini",
		// OpenAI — GPT-4.1
		"gpt-4.1",
		"gpt-4.1-mini",
		"gpt-4.1-nano",
		// Anthropic 4.5/4.6
		"claude-opus-4.6",
		"claude-sonnet-4.6",
		"claude-haiku-4.5",
		// Anthropic — dated variants (prefix match)
		"claude-opus-4.6-20260101",
		"claude-sonnet-4.6-20260101",
		"claude-haiku-4.5-20251001",
		// Anthropic — thinking variants (prefix match)
		"claude-opus-4.6-thinking",
		"claude-sonnet-4.5-thinking",
		// xAI
		"grok-3",
		"grok-3-mini",
	}

	for _, model := range models {
		if !m.KnownModel(model) {
			t.Errorf("%q should be a known model", model)
		}
	}
}

func TestO3ProNotConfusedWithO3(t *testing.T) {
	m := New()

	// o3-pro must NOT use o3 pricing ($10/$40), it should use its own ($150/$600)
	cost := m.Calculate("o3-pro", 1_000_000, 0)
	// o3-pro input: $150 per MTok
	if math.Abs(cost-150.00) > 1e-9 {
		t.Errorf("o3-pro: expected $150.00 input cost, got $%f (may have matched o3 instead)", cost)
	}
}
