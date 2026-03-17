package provider

// NewDeepSeek creates a DeepSeek provider. DeepSeek uses the OpenAI-compatible API format.
// Requests arrive at /deepseek/v1/chat/completions and are proxied to the DeepSeek API.
func NewDeepSeek(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://api.deepseek.com"
	}
	return NewOpenAICompatible("deepseek", upstream, "/deepseek")
}
