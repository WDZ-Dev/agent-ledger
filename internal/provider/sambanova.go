package provider

// NewSambaNova creates a SambaNova provider. SambaNova uses the OpenAI-compatible
// API format. Requests arrive at /sambanova/v1/chat/completions.
func NewSambaNova(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://api.sambanova.ai"
	}
	return NewOpenAICompatible("sambanova", upstream, "/sambanova")
}
