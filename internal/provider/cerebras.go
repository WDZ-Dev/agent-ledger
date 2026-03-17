package provider

// NewCerebras creates a Cerebras provider. Cerebras uses the OpenAI-compatible
// API format. Requests arrive at /cerebras/v1/chat/completions.
func NewCerebras(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://api.cerebras.ai"
	}
	return NewOpenAICompatible("cerebras", upstream, "/cerebras")
}
