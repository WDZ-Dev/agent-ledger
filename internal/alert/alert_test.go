package alert

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type countingNotifier struct {
	count atomic.Int64
}

func (c *countingNotifier) Notify(_ context.Context, _ Alert) error {
	c.count.Add(1)
	return nil
}

func TestMultiNotifier(t *testing.T) {
	n1 := &countingNotifier{}
	n2 := &countingNotifier{}
	multi := NewMultiNotifier(n1, n2)

	if err := multi.Notify(context.Background(), Alert{Type: "test"}); err != nil {
		t.Fatal(err)
	}
	if n1.count.Load() != 1 {
		t.Errorf("n1 count = %d, want 1", n1.count.Load())
	}
	if n2.count.Load() != 1 {
		t.Errorf("n2 count = %d, want 1", n2.count.Load())
	}
}

func TestRateLimitedNotifier(t *testing.T) {
	inner := &countingNotifier{}
	rl := NewRateLimitedNotifier(inner, 100*time.Millisecond)

	a := Alert{Type: "test", Details: map[string]string{"session_id": "s1"}}

	// First call should go through.
	if err := rl.Notify(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if inner.count.Load() != 1 {
		t.Errorf("count = %d, want 1", inner.count.Load())
	}

	// Second call within cooldown should be suppressed.
	if err := rl.Notify(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if inner.count.Load() != 1 {
		t.Errorf("count = %d, want 1 (suppressed)", inner.count.Load())
	}

	// Different alert key should go through.
	a2 := Alert{Type: "test", Details: map[string]string{"session_id": "s2"}}
	if err := rl.Notify(context.Background(), a2); err != nil {
		t.Fatal(err)
	}
	if inner.count.Load() != 2 {
		t.Errorf("count = %d, want 2", inner.count.Load())
	}

	// After cooldown, same alert should go through again.
	time.Sleep(150 * time.Millisecond)
	if err := rl.Notify(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if inner.count.Load() != 3 {
		t.Errorf("count = %d, want 3 (after cooldown)", inner.count.Load())
	}
}
