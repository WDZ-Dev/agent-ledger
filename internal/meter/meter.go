package meter

import "strings"

// Meter calculates costs from token usage and model pricing.
type Meter struct {
	pricing map[string]ModelPricing
}

// New creates a Meter with the default pricing table.
func New() *Meter {
	return &Meter{pricing: DefaultPricing()}
}

// Calculate returns the cost in USD for the given token usage.
// Returns 0 if the model is unknown.
func (m *Meter) Calculate(model string, inputTokens, outputTokens int) float64 {
	p, ok := m.findPricing(model)
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) / 1_000_000 * p.InputPerMTok
	outputCost := float64(outputTokens) / 1_000_000 * p.OutputPerMTok
	return inputCost + outputCost
}

// KnownModel returns true if the model has pricing information.
func (m *Meter) KnownModel(model string) bool {
	_, ok := m.findPricing(model)
	return ok
}

// findPricing looks up pricing for a model. It tries exact match first,
// then longest prefix match to handle versioned model names like
// "gpt-4o-2024-11-20".
func (m *Meter) findPricing(model string) (ModelPricing, bool) {
	if p, ok := m.pricing[model]; ok {
		return p, true
	}

	var bestKey string
	var bestPricing ModelPricing
	for key, p := range m.pricing {
		if strings.HasPrefix(model, key) && len(key) > len(bestKey) {
			bestKey = key
			bestPricing = p
		}
	}
	if bestKey != "" {
		return bestPricing, true
	}

	return ModelPricing{}, false
}
