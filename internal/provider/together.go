package provider

// NewTogether creates a Together AI provider. Together uses the OpenAI-compatible
// API format. Requests arrive at /together/v1/chat/completions.
func NewTogether(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://api.together.xyz"
	}
	return NewOpenAICompatible("together", upstream, "/together")
}
