package provider

import (
	"net/http"

	"github.com/WDZ-Dev/agent-ledger/internal/config"
)

// Registry holds configured providers and detects which one should handle
// a given request.
type Registry struct {
	providers []Provider
}

// NewRegistry creates a Registry from configuration, including only
// enabled providers.
func NewRegistry(cfg config.ProvidersConfig) *Registry {
	var providers []Provider

	if cfg.OpenAI.Enabled {
		providers = append(providers, NewOpenAI(cfg.OpenAI.Upstream))
	}
	if cfg.Anthropic.Enabled {
		providers = append(providers, NewAnthropic(cfg.Anthropic.Upstream))
	}

	return &Registry{providers: providers}
}

// Detect returns the first provider that matches the request, or nil.
func (r *Registry) Detect(req *http.Request) Provider {
	for _, p := range r.providers {
		if p.Match(req) {
			return p
		}
	}
	return nil
}
