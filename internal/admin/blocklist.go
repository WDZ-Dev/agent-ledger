package admin

import (
	"context"
	"path/filepath"
	"sync"
	"time"
)

// Blocklist checks API keys against a list of blocked glob patterns.
type Blocklist struct {
	store    *Store
	patterns []string
	mu       sync.RWMutex
	lastLoad time.Time
	ttl      time.Duration
}

// NewBlocklist creates a Blocklist backed by the admin config store.
func NewBlocklist(store *Store) *Blocklist {
	return &Blocklist{
		store: store,
		ttl:   10 * time.Second,
	}
}

// IsBlocked returns true if the raw API key matches any blocked pattern.
func (b *Blocklist) IsBlocked(rawKey string) bool {
	b.mu.RLock()
	patterns := b.patterns
	lastLoad := b.lastLoad
	b.mu.RUnlock()

	if time.Since(lastLoad) > b.ttl {
		b.refresh()
		b.mu.RLock()
		patterns = b.patterns
		b.mu.RUnlock()
	}

	for _, p := range patterns {
		if matched, _ := filepath.Match(p, rawKey); matched {
			return true
		}
	}
	return false
}

// Refresh reloads blocked patterns from the store.
func (b *Blocklist) Refresh() {
	b.refresh()
}

func (b *Blocklist) refresh() {
	var patterns []string
	_ = b.store.GetJSON(context.Background(), "blocked_keys", &patterns)
	b.mu.Lock()
	b.patterns = patterns
	b.lastLoad = time.Now()
	b.mu.Unlock()
}
