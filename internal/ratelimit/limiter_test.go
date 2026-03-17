package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllows(t *testing.T) {
	l := New(Config{
		Default: Rule{RequestsPerMinute: 3},
	})

	for i := 0; i < 3; i++ {
		ok, _ := l.Allow("sk-test", "hash1")
		if !ok {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Fourth request should be blocked.
	ok, retryAfter := l.Allow("sk-test", "hash1")
	if ok {
		t.Error("fourth request should be rate limited")
	}
	if retryAfter <= 0 {
		t.Error("retry-after should be positive")
	}
}

func TestLimiterEnabled(t *testing.T) {
	l := New(Config{})
	if l.Enabled() {
		t.Error("should not be enabled with zero config")
	}

	l = New(Config{Default: Rule{RequestsPerMinute: 10}})
	if !l.Enabled() {
		t.Error("should be enabled with default rule")
	}
}

func TestLimiterPerKeyRules(t *testing.T) {
	l := New(Config{
		Default: Rule{RequestsPerMinute: 100},
		Rules: []Rule{
			{APIKeyPattern: "sk-dev-*", RequestsPerMinute: 2},
		},
	})

	// Dev key — limited to 2/min.
	l.Allow("sk-dev-abc", "dev-hash")
	l.Allow("sk-dev-abc", "dev-hash")
	ok, _ := l.Allow("sk-dev-abc", "dev-hash")
	if ok {
		t.Error("dev key should be rate limited at 2 req/min")
	}

	// Prod key — 100/min, should be fine.
	ok, _ = l.Allow("sk-prod-xyz", "prod-hash")
	if !ok {
		t.Error("prod key should be allowed")
	}
}

func TestLimiterWindowResets(t *testing.T) {
	l := New(Config{
		Default: Rule{RequestsPerMinute: 1},
	})

	ok, _ := l.Allow("sk-test", "hash1")
	if !ok {
		t.Error("first request should be allowed")
	}
	ok, _ = l.Allow("sk-test", "hash1")
	if ok {
		t.Error("second request should be blocked")
	}

	// Simulate window expiry by manipulating the counter.
	l.mu.Lock()
	l.minuteCounters["hash1"].windowEnd = time.Now().Add(-time.Second)
	l.mu.Unlock()

	ok, _ = l.Allow("sk-test", "hash1")
	if !ok {
		t.Error("should be allowed after window reset")
	}
}
