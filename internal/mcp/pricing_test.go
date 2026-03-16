package mcp

import "testing"

func TestPricer_ExactMatch(t *testing.T) {
	p := NewPricer([]PricingRule{
		{Server: "filesystem", Tool: "read_file", CostPerCall: 0.01},
		{Server: "filesystem", Tool: "write_file", CostPerCall: 0.05},
	})

	if got := p.CostForCall("filesystem", "read_file"); got != 0.01 {
		t.Errorf("cost = %f, want 0.01", got)
	}
	if got := p.CostForCall("filesystem", "write_file"); got != 0.05 {
		t.Errorf("cost = %f, want 0.05", got)
	}
}

func TestPricer_ServerWildcard(t *testing.T) {
	p := NewPricer([]PricingRule{
		{Server: "github", Tool: "", CostPerCall: 0.02},
	})

	if got := p.CostForCall("github", "create_issue"); got != 0.02 {
		t.Errorf("cost = %f, want 0.02", got)
	}
	if got := p.CostForCall("github", "list_prs"); got != 0.02 {
		t.Errorf("cost = %f, want 0.02", got)
	}
}

func TestPricer_ExactTakesPrecedence(t *testing.T) {
	p := NewPricer([]PricingRule{
		{Server: "db", Tool: "", CostPerCall: 0.01},
		{Server: "db", Tool: "execute_query", CostPerCall: 0.10},
	})

	if got := p.CostForCall("db", "execute_query"); got != 0.10 {
		t.Errorf("exact match cost = %f, want 0.10", got)
	}
	if got := p.CostForCall("db", "list_tables"); got != 0.01 {
		t.Errorf("wildcard cost = %f, want 0.01", got)
	}
}

func TestPricer_NoMatch(t *testing.T) {
	p := NewPricer([]PricingRule{
		{Server: "filesystem", Tool: "read_file", CostPerCall: 0.01},
	})

	if got := p.CostForCall("unknown", "read_file"); got != 0.0 {
		t.Errorf("cost = %f, want 0.0 for unknown server", got)
	}
}

func TestPricer_EmptyRules(t *testing.T) {
	p := NewPricer(nil)

	if got := p.CostForCall("anything", "anything"); got != 0.0 {
		t.Errorf("cost = %f, want 0.0 for empty rules", got)
	}
}
