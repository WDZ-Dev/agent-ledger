package budget

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is open and rejecting requests.
var ErrCircuitOpen = errors.New("circuit breaker is open: upstream unavailable")

type breakerState int32

const (
	stateClosed   breakerState = iota // normal operation
	stateOpen                         // rejecting requests
	stateHalfOpen                     // testing with limited requests
)

// BreakerConfig holds circuit breaker settings.
type BreakerConfig struct {
	MaxFailures int   `mapstructure:"max_failures"`
	TimeoutSecs int64 `mapstructure:"timeout_secs"`
}

// BreakerTransport wraps an http.RoundTripper with circuit breaker protection
// against upstream failures. After MaxFailures consecutive 5xx responses or
// transport errors, the circuit opens and rejects requests for TimeoutSecs.
type BreakerTransport struct {
	transport   http.RoundTripper
	state       atomic.Int32
	failures    atomic.Int64
	maxFailures int64
	timeout     time.Duration
	lastTripped time.Time
	mu          sync.Mutex
}

// NewBreakerTransport creates a circuit-breaker-protected transport.
func NewBreakerTransport(transport http.RoundTripper, maxFailures int64, timeout time.Duration) *BreakerTransport {
	return &BreakerTransport{
		transport:   transport,
		maxFailures: maxFailures,
		timeout:     timeout,
	}
}

// RoundTrip executes the request with circuit breaker protection.
func (bt *BreakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !bt.allow() {
		return nil, ErrCircuitOpen
	}

	resp, err := bt.transport.RoundTrip(req)
	if err != nil {
		bt.recordFailure()
		return nil, err
	}

	if resp.StatusCode >= 500 {
		bt.recordFailure()
	} else {
		bt.recordSuccess()
	}

	return resp, nil
}

func (bt *BreakerTransport) allow() bool {
	switch breakerState(bt.state.Load()) {
	case stateOpen:
		bt.mu.Lock()
		elapsed := time.Since(bt.lastTripped)
		bt.mu.Unlock()
		if elapsed > bt.timeout {
			// Transition to half-open: allow one test request.
			bt.state.CompareAndSwap(int32(stateOpen), int32(stateHalfOpen))
			return true
		}
		return false
	default:
		return true
	}
}

func (bt *BreakerTransport) recordFailure() {
	count := bt.failures.Add(1)
	if count >= bt.maxFailures {
		bt.mu.Lock()
		bt.state.Store(int32(stateOpen))
		bt.lastTripped = time.Now()
		bt.mu.Unlock()
	}
}

func (bt *BreakerTransport) recordSuccess() {
	bt.failures.Store(0)
	bt.state.Store(int32(stateClosed))
}

const (
	stateNameClosed   = "closed"
	stateNameOpen     = "open"
	stateNameHalfOpen = "half-open"
)

// State returns the current circuit breaker state as a string.
func (bt *BreakerTransport) State() string {
	switch breakerState(bt.state.Load()) {
	case stateOpen:
		return stateNameOpen
	case stateHalfOpen:
		return stateNameHalfOpen
	default:
		return stateNameClosed
	}
}
