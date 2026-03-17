package tenant

import (
	"net/http"
	"path/filepath"
)

// Resolver determines the tenant for a given request.
type Resolver interface {
	ResolveTenant(r *http.Request) string
}

// HeaderResolver reads the tenant ID from the X-AgentLedger-Tenant header.
type HeaderResolver struct{}

func (h *HeaderResolver) ResolveTenant(r *http.Request) string {
	return r.Header.Get("X-AgentLedger-Tenant")
}

// KeyMapping maps an API key glob pattern to a tenant ID.
type KeyMapping struct {
	APIKeyPattern string `mapstructure:"api_key_pattern"`
	TenantID      string `mapstructure:"tenant_id"`
}

// ConfigResolver maps API keys to tenants using glob patterns from config.
type ConfigResolver struct {
	mappings []KeyMapping
}

// NewConfigResolver creates a resolver from API key → tenant mappings.
func NewConfigResolver(mappings []KeyMapping) *ConfigResolver {
	return &ConfigResolver{mappings: mappings}
}

// ResolveTenant checks the raw API key against configured glob patterns.
// It reads the key from the same sources as provider.ExtractAPIKey.
func (c *ConfigResolver) ResolveTenant(r *http.Request) string {
	key := extractRawKey(r)
	if key == "" {
		return ""
	}
	for _, m := range c.mappings {
		if matched, _ := filepath.Match(m.APIKeyPattern, key); matched {
			return m.TenantID
		}
	}
	return ""
}

func extractRawKey(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	if key := r.Header.Get("x-api-key"); key != "" {
		return key
	}
	if key := r.Header.Get("X-Goog-Api-Key"); key != "" {
		return key
	}
	return ""
}

// ChainResolver tries multiple resolvers in order, returning the first
// non-empty tenant ID.
type ChainResolver struct {
	resolvers []Resolver
}

// NewChainResolver creates a resolver that chains multiple resolvers.
func NewChainResolver(resolvers ...Resolver) *ChainResolver {
	return &ChainResolver{resolvers: resolvers}
}

func (c *ChainResolver) ResolveTenant(r *http.Request) string {
	for _, res := range c.resolvers {
		if id := res.ResolveTenant(r); id != "" {
			return id
		}
	}
	return ""
}
