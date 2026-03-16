package provider

// NewMistral creates a Mistral provider. Mistral uses the OpenAI-compatible API format.
// Requests arrive at /mistral/v1/chat/completions and are proxied to the Mistral API.
func NewMistral(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://api.mistral.ai"
	}
	return NewOpenAICompatible("mistral", upstream, "/mistral")
}
