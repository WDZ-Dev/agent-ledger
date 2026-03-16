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
// enabled providers. Extra providers from config are instantiated by type.
func NewRegistry(cfg config.ProvidersConfig) *Registry {
	var providers []Provider

	if cfg.OpenAI.Enabled {
		providers = append(providers, NewOpenAI(cfg.OpenAI.Upstream))
	}
	if cfg.Anthropic.Enabled {
		providers = append(providers, NewAnthropic(cfg.Anthropic.Upstream))
	}

	// Dynamic providers from config.Extra.
	for name, pc := range cfg.Extra {
		if !pc.Enabled {
			continue
		}
		p := NewProviderByType(name, pc.Type, pc.Upstream, pc.PathPrefix)
		if p != nil {
			providers = append(providers, p)
		}
	}

	return &Registry{providers: providers}
}

// NewProviderByType creates a provider by its type string.
func NewProviderByType(name, typ, upstream, pathPrefix string) Provider {
	switch typ {
	case "openai":
		if upstream == "" {
			upstream = "https://api.openai.com"
		}
		if pathPrefix == "" {
			pathPrefix = "/" + name
		}
		return NewOpenAICompatible(name, upstream, pathPrefix)
	case "anthropic":
		if upstream == "" {
			upstream = "https://api.anthropic.com"
		}
		return NewAnthropic(upstream)
	case "gemini": //nolint:goconst
		return NewGemini(upstream)
	case "cohere": //nolint:goconst
		return NewCohere(upstream)
	default:
		// Unknown type — try as OpenAI-compatible with a prefix.
		if upstream != "" {
			if pathPrefix == "" {
				pathPrefix = "/" + name
			}
			return NewOpenAICompatible(name, upstream, pathPrefix)
		}
		return nil
	}
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

// PathPrefixes returns the path prefixes of all registered providers that
// have a non-empty prefix (used for registering mux routes).
func (r *Registry) PathPrefixes() []string {
	seen := map[string]bool{}
	var prefixes []string
	for _, p := range r.providers {
		type prefixer interface {
			PathPrefix() string
		}
		if pp, ok := p.(prefixer); ok {
			prefix := pp.PathPrefix()
			if prefix != "" && !seen[prefix] {
				seen[prefix] = true
				prefixes = append(prefixes, prefix)
			}
		}
	}
	return prefixes
}
