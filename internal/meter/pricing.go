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
		// OpenAI — GPT-4.1 family
		"gpt-4.1":      {InputPerMTok: 2.00, OutputPerMTok: 8.00},
		"gpt-4.1-mini": {InputPerMTok: 0.40, OutputPerMTok: 1.60},
		"gpt-4.1-nano": {InputPerMTok: 0.10, OutputPerMTok: 0.40},

		// OpenAI — GPT-4o family
		"gpt-4o":      {InputPerMTok: 2.50, OutputPerMTok: 10.00},
		"gpt-4o-mini": {InputPerMTok: 0.15, OutputPerMTok: 0.60},

		// OpenAI — reasoning models
		"o3":      {InputPerMTok: 10.00, OutputPerMTok: 40.00},
		"o3-mini": {InputPerMTok: 1.10, OutputPerMTok: 4.40},
		"o4-mini": {InputPerMTok: 1.10, OutputPerMTok: 4.40},
		"o1":      {InputPerMTok: 15.00, OutputPerMTok: 60.00},
		"o1-mini": {InputPerMTok: 3.00, OutputPerMTok: 12.00},

		// OpenAI — legacy
		"gpt-4-turbo":   {InputPerMTok: 10.00, OutputPerMTok: 30.00},
		"gpt-4":         {InputPerMTok: 30.00, OutputPerMTok: 60.00},
		"gpt-3.5-turbo": {InputPerMTok: 0.50, OutputPerMTok: 1.50},

		// Anthropic — Claude 4 family
		"claude-opus-4":   {InputPerMTok: 15.00, OutputPerMTok: 75.00},
		"claude-sonnet-4": {InputPerMTok: 3.00, OutputPerMTok: 15.00},
		"claude-haiku-4":  {InputPerMTok: 0.80, OutputPerMTok: 4.00},

		// Anthropic — Claude 3.5
		"claude-3-5-sonnet": {InputPerMTok: 3.00, OutputPerMTok: 15.00},
		"claude-3-5-haiku":  {InputPerMTok: 0.80, OutputPerMTok: 4.00},

		// Anthropic — Claude 3
		"claude-3-opus":   {InputPerMTok: 15.00, OutputPerMTok: 75.00},
		"claude-3-sonnet": {InputPerMTok: 3.00, OutputPerMTok: 15.00},
		"claude-3-haiku":  {InputPerMTok: 0.25, OutputPerMTok: 1.25},

		// Google Gemini
		"gemini-2.5-pro":   {InputPerMTok: 1.25, OutputPerMTok: 10.00},
		"gemini-2.5-flash": {InputPerMTok: 0.15, OutputPerMTok: 0.60},
		"gemini-2.0-flash": {InputPerMTok: 0.10, OutputPerMTok: 0.40},
		"gemini-1.5-pro":   {InputPerMTok: 1.25, OutputPerMTok: 5.00},
		"gemini-1.5-flash": {InputPerMTok: 0.075, OutputPerMTok: 0.30},

		// Mistral
		"mistral-large-latest": {InputPerMTok: 2.00, OutputPerMTok: 6.00},
		"mistral-small-latest": {InputPerMTok: 0.20, OutputPerMTok: 0.60},
		"codestral-latest":     {InputPerMTok: 0.30, OutputPerMTok: 0.90},
		"open-mistral-nemo":    {InputPerMTok: 0.15, OutputPerMTok: 0.15},

		// Groq (hosted models — pricing reflects Groq's rates)
		"llama-3.3-70b-versatile": {InputPerMTok: 0.59, OutputPerMTok: 0.79},
		"llama-3.1-8b-instant":    {InputPerMTok: 0.05, OutputPerMTok: 0.08},
		"mixtral-8x7b-32768":      {InputPerMTok: 0.24, OutputPerMTok: 0.24},
		"gemma2-9b-it":            {InputPerMTok: 0.20, OutputPerMTok: 0.20},

		// DeepSeek
		"deepseek-chat":     {InputPerMTok: 0.14, OutputPerMTok: 0.28},
		"deepseek-reasoner": {InputPerMTok: 0.55, OutputPerMTok: 2.19},

		// Cohere
		"command-r-plus": {InputPerMTok: 2.50, OutputPerMTok: 10.00},
		"command-r":      {InputPerMTok: 0.15, OutputPerMTok: 0.60},
		"command-light":  {InputPerMTok: 0.30, OutputPerMTok: 0.60},
	}
}
