package provider

// NewXAI creates an xAI (Grok) provider. xAI uses the OpenAI-compatible
// API format. Requests arrive at /xai/v1/chat/completions.
func NewXAI(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://api.x.ai"
	}
	return NewOpenAICompatible("xai", upstream, "/xai")
}
