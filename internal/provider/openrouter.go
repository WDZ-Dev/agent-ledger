package provider

// NewOpenRouter creates an OpenRouter provider. OpenRouter uses the OpenAI-compatible
// API format to route to many models. Requests arrive at /openrouter/v1/chat/completions.
func NewOpenRouter(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://openrouter.ai/api"
	}
	return NewOpenAICompatible("openrouter", upstream, "/openrouter")
}
