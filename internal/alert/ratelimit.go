package alert

import (
	"context"
	"sync"
	"time"
)

// RateLimitedNotifier wraps a notifier with per-alert-key deduplication.
// The same alert key (type + first detail value) is suppressed for the
// cooldown duration.
type RateLimitedNotifier struct {
	inner    Notifier
	cooldown time.Duration
	mu       sync.Mutex
	seen     map[string]time.Time
}

// NewRateLimitedNotifier wraps a notifier with deduplication cooldown.
func NewRateLimitedNotifier(inner Notifier, cooldown time.Duration) *RateLimitedNotifier {
	return &RateLimitedNotifier{
		inner:    inner,
		cooldown: cooldown,
		seen:     make(map[string]time.Time),
	}
}

func (r *RateLimitedNotifier) Notify(ctx context.Context, a Alert) error {
	key := alertKey(a)

	r.mu.Lock()
	if lastSent, ok := r.seen[key]; ok && time.Since(lastSent) < r.cooldown {
		r.mu.Unlock()
		return nil // suppressed
	}
	r.seen[key] = time.Now()

	// Evict old entries.
	for k, t := range r.seen {
		if time.Since(t) > r.cooldown*2 {
			delete(r.seen, k)
		}
	}
	r.mu.Unlock()

	return r.inner.Notify(ctx, a)
}

func alertKey(a Alert) string {
	key := a.Type
	// Use session_id or api_key_hash as scope if available.
	for _, k := range []string{"session_id", "api_key_hash"} {
		if v, ok := a.Details[k]; ok {
			key += ":" + v
			break
		}
	}
	return key
}
