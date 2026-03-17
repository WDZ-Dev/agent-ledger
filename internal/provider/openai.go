package provider

// NewOpenAI creates the OpenAI provider. It is a thin wrapper over
// OpenAICompatible with no path prefix (requests arrive at /v1/...).
func NewOpenAI(upstream string) *OpenAICompatible {
	return NewOpenAICompatible("openai", upstream, "")
}
