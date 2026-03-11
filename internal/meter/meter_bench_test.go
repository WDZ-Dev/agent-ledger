package meter

import "testing"

func BenchmarkCalculate(b *testing.B) {
	m := New()
	b.ResetTimer()
	for range b.N {
		m.Calculate("gpt-4o-mini-2024-07-18", 1000, 500)
	}
}

func BenchmarkEstimateTokens(b *testing.B) {
	m := New()
	text := "The quick brown fox jumps over the lazy dog. This is a test of the tiktoken estimation."

	// Warm up encoder cache.
	m.EstimateTokens("gpt-4o", text)

	b.ResetTimer()
	for range b.N {
		m.EstimateTokens("gpt-4o", text)
	}
}
