package provider

// NewFireworks creates a Fireworks AI provider. Fireworks uses the OpenAI-compatible
// API format. Requests arrive at /fireworks/v1/chat/completions.
func NewFireworks(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://api.fireworks.ai/inference"
	}
	return NewOpenAICompatible("fireworks", upstream, "/fireworks")
}
