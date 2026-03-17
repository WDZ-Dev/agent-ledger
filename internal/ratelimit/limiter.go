package ratelimit

import (
	"path/filepath"
	"sync"
	"time"
)

// Rule defines rate limits for a set of API keys.
type Rule struct {
	APIKeyPattern     string `mapstructure:"api_key_pattern"`
	RequestsPerMinute int    `mapstructure:"requests_per_minute"`
	RequestsPerHour   int    `mapstructure:"requests_per_hour"`
}

// Config holds rate limiting configuration.
type Config struct {
	Default Rule   `mapstructure:"default"`
	Rules   []Rule `mapstructure:"rules"`
}

// Limiter enforces request rate limits using a sliding window counter.
type Limiter struct {
	config Config
	mu     sync.Mutex
	// key -> window start -> count
	minuteCounters map[string]*slidingWindow
	hourCounters   map[string]*slidingWindow
}

type slidingWindow struct {
	count     int
	windowEnd time.Time
}

// New creates a rate limiter from configuration.
func New(cfg Config) *Limiter {
	return &Limiter{
		config:         cfg,
		minuteCounters: make(map[string]*slidingWindow),
		hourCounters:   make(map[string]*slidingWindow),
	}
}

// Enabled returns true if any rate limits are configured.
func (l *Limiter) Enabled() bool {
	return l.config.Default.RequestsPerMinute > 0 ||
		l.config.Default.RequestsPerHour > 0 ||
		len(l.config.Rules) > 0
}

// Allow checks if a request is allowed for the given API key.
// Returns allowed, retryAfter duration.
func (l *Limiter) Allow(rawKey, apiKeyHash string) (bool, time.Duration) {
	rule := l.matchRule(rawKey)
	if rule.RequestsPerMinute <= 0 && rule.RequestsPerHour <= 0 {
		return true, 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Check minute limit.
	if rule.RequestsPerMinute > 0 {
		w := l.getOrCreateWindow(l.minuteCounters, apiKeyHash, now, time.Minute)
		if w.count >= rule.RequestsPerMinute {
			retryAfter := w.windowEnd.Sub(now)
			return false, retryAfter
		}
	}

	// Check hour limit.
	if rule.RequestsPerHour > 0 {
		w := l.getOrCreateWindow(l.hourCounters, apiKeyHash, now, time.Hour)
		if w.count >= rule.RequestsPerHour {
			retryAfter := w.windowEnd.Sub(now)
			return false, retryAfter
		}
	}

	// Increment counters.
	if rule.RequestsPerMinute > 0 {
		l.minuteCounters[apiKeyHash].count++
	}
	if rule.RequestsPerHour > 0 {
		l.hourCounters[apiKeyHash].count++
	}

	return true, 0
}

func (l *Limiter) getOrCreateWindow(counters map[string]*slidingWindow, key string, now time.Time, duration time.Duration) *slidingWindow {
	w, ok := counters[key]
	if !ok || now.After(w.windowEnd) {
		w = &slidingWindow{
			count:     0,
			windowEnd: now.Add(duration),
		}
		counters[key] = w
	}
	return w
}

func (l *Limiter) matchRule(rawKey string) Rule {
	for _, r := range l.config.Rules {
		if matched, _ := filepath.Match(r.APIKeyPattern, rawKey); matched {
			return l.mergeWithDefault(r)
		}
	}
	return l.config.Default
}

func (l *Limiter) mergeWithDefault(r Rule) Rule {
	if r.RequestsPerMinute <= 0 {
		r.RequestsPerMinute = l.config.Default.RequestsPerMinute
	}
	if r.RequestsPerHour <= 0 {
		r.RequestsPerHour = l.config.Default.RequestsPerHour
	}
	return r
}
