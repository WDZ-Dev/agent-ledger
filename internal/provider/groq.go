package provider

// NewGroq creates a Groq provider. Groq uses the OpenAI-compatible API format.
// Requests arrive at /groq/v1/chat/completions and are proxied to the Groq API.
func NewGroq(upstream string) *OpenAICompatible {
	if upstream == "" {
		upstream = "https://api.groq.com/openai"
	}
	return NewOpenAICompatible("groq", upstream, "/groq")
}
