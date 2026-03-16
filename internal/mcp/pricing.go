package mcp

// PricingRule maps a server+tool combination to a per-call cost.
// An empty Tool field acts as a wildcard for all tools on that server.
type PricingRule struct {
	Server      string
	Tool        string
	CostPerCall float64
}

// Pricer resolves per-call costs for MCP tool invocations.
type Pricer struct {
	rules []PricingRule
}

// NewPricer creates a Pricer from the given rules.
func NewPricer(rules []PricingRule) *Pricer {
	return &Pricer{rules: rules}
}

// CostForCall returns the cost for a single tool call.
// Match precedence: exact server+tool > server wildcard (empty tool) > 0.0.
func (p *Pricer) CostForCall(serverName, toolName string) float64 {
	var wildcardCost float64
	wildcardFound := false

	for _, r := range p.rules {
		if r.Server != serverName {
			continue
		}
		if r.Tool == toolName {
			return r.CostPerCall
		}
		if r.Tool == "" {
			wildcardCost = r.CostPerCall
			wildcardFound = true
		}
	}

	if wildcardFound {
		return wildcardCost
	}
	return 0.0
}
