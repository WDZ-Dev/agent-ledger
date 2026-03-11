package meter

// ModelPricing holds per-model token costs in USD per million tokens.
type ModelPricing struct {
	InputPerMTok  float64
	OutputPerMTok float64
}

// DefaultPricing returns the built-in pricing table for known models.
// Prices are in USD per million tokens.
func DefaultPricing() map[string]ModelPricing {
	return map[string]ModelPricing{
		// OpenAI
		"gpt-4o":        {InputPerMTok: 2.50, OutputPerMTok: 10.00},
		"gpt-4o-mini":   {InputPerMTok: 0.15, OutputPerMTok: 0.60},
		"gpt-4-turbo":   {InputPerMTok: 10.00, OutputPerMTok: 30.00},
		"gpt-4":         {InputPerMTok: 30.00, OutputPerMTok: 60.00},
		"gpt-3.5-turbo": {InputPerMTok: 0.50, OutputPerMTok: 1.50},
		"o1":            {InputPerMTok: 15.00, OutputPerMTok: 60.00},
		"o1-mini":       {InputPerMTok: 3.00, OutputPerMTok: 12.00},
		"o3":            {InputPerMTok: 10.00, OutputPerMTok: 40.00},
		"o3-mini":       {InputPerMTok: 1.10, OutputPerMTok: 4.40},
		"o4-mini":       {InputPerMTok: 1.10, OutputPerMTok: 4.40},

		// Anthropic
		"claude-opus-4":     {InputPerMTok: 15.00, OutputPerMTok: 75.00},
		"claude-sonnet-4":   {InputPerMTok: 3.00, OutputPerMTok: 15.00},
		"claude-3-5-sonnet": {InputPerMTok: 3.00, OutputPerMTok: 15.00},
		"claude-3-5-haiku":  {InputPerMTok: 0.80, OutputPerMTok: 4.00},
		"claude-3-opus":     {InputPerMTok: 15.00, OutputPerMTok: 75.00},
		"claude-3-sonnet":   {InputPerMTok: 3.00, OutputPerMTok: 15.00},
		"claude-3-haiku":    {InputPerMTok: 0.25, OutputPerMTok: 1.25},
	}
}
