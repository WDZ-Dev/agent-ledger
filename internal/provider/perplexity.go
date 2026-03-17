package provider

// NewPerplexity creates a Perplexity provider. Perplexity uses the OpenAI-compatible
// API format. Requests arrive at /perplexity/v1/chat/completions.
func NewPerplexity(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://api.perplexity.ai"
	}
	return NewOpenAICompatible("perplexity", upstream, "/perplexity")
}
