package tenant

import (
	"net/http"
	"testing"
)

func TestHeaderResolver(t *testing.T) {
	h := &HeaderResolver{}

	r := &http.Request{Header: http.Header{
		"X-Agentledger-Tenant": {"team-alpha"},
	}}
	if got := h.ResolveTenant(r); got != "team-alpha" {
		t.Errorf("got %q, want %q", got, "team-alpha")
	}

	r = &http.Request{Header: http.Header{}}
	if got := h.ResolveTenant(r); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestConfigResolver(t *testing.T) {
	c := NewConfigResolver([]KeyMapping{
		{APIKeyPattern: "sk-proj-dev-*", TenantID: "dev-team"},
		{APIKeyPattern: "sk-proj-prod-*", TenantID: "prod-team"},
	})

	tests := []struct {
		key  string
		want string
	}{
		{"sk-proj-dev-abc123", "dev-team"},
		{"sk-proj-prod-xyz789", "prod-team"},
		{"sk-proj-other-key", ""},
	}

	for _, tt := range tests {
		r := &http.Request{Header: http.Header{
			"Authorization": {"Bearer " + tt.key},
		}}
		if got := c.ResolveTenant(r); got != tt.want {
			t.Errorf("key=%q: got %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestChainResolver(t *testing.T) {
	header := &HeaderResolver{}
	config := NewConfigResolver([]KeyMapping{
		{APIKeyPattern: "sk-dev-*", TenantID: "from-config"},
	})
	chain := NewChainResolver(header, config)

	// Header takes priority.
	r := &http.Request{Header: http.Header{
		"X-Agentledger-Tenant": {"from-header"},
		"Authorization":        {"Bearer sk-dev-abc"},
	}}
	if got := chain.ResolveTenant(r); got != "from-header" {
		t.Errorf("got %q, want %q", got, "from-header")
	}

	// Falls back to config resolver.
	r = &http.Request{Header: http.Header{
		"Authorization": {"Bearer sk-dev-abc"},
	}}
	if got := chain.ResolveTenant(r); got != "from-config" {
		t.Errorf("got %q, want %q", got, "from-config")
	}

	// Neither matches — empty.
	r = &http.Request{Header: http.Header{
		"Authorization": {"Bearer sk-other-key"},
	}}
	if got := chain.ResolveTenant(r); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
