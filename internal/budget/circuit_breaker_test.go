package budget

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestBreakerClosedOnSuccess(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	bt := NewBreakerTransport(http.DefaultTransport, 3, 5*time.Second)

	req, _ := http.NewRequest("GET", upstream.URL, nil)
	resp, err := bt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if bt.State() != stateNameClosed {
		t.Errorf("state = %q, want closed", bt.State())
	}
}

func TestBreakerOpensAfterFailures(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer upstream.Close()

	bt := NewBreakerTransport(http.DefaultTransport, 3, 5*time.Second)

	for range 3 {
		req, _ := http.NewRequest("GET", upstream.URL, nil)
		resp, err := bt.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}

	if bt.State() != stateNameOpen {
		t.Errorf("state = %q, want open after 3 failures", bt.State())
	}

	// Next request should be rejected.
	req, _ := http.NewRequest("GET", upstream.URL, nil)
	resp, err := bt.RoundTrip(req)
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
}

func TestBreakerRecoverAfterTimeout(t *testing.T) {
	var status atomic.Int32
	status.Store(500)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(int(status.Load()))
	}))
	defer upstream.Close()

	bt := NewBreakerTransport(http.DefaultTransport, 2, 100*time.Millisecond)

	// Trip the breaker.
	for range 2 {
		req, _ := http.NewRequest("GET", upstream.URL, nil)
		resp, _ := bt.RoundTrip(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	if bt.State() != stateNameOpen {
		t.Fatal("expected open state")
	}

	// Wait for timeout.
	time.Sleep(150 * time.Millisecond)

	// Fix upstream.
	status.Store(200)

	req, _ := http.NewRequest("GET", upstream.URL, nil)
	resp, err := bt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected recovery, got error: %v", err)
	}
	_ = resp.Body.Close()

	if bt.State() != stateNameClosed {
		t.Errorf("state = %q, want closed after recovery", bt.State())
	}
}

func TestBreakerPassesResponses(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer upstream.Close()

	bt := NewBreakerTransport(http.DefaultTransport, 5, 5*time.Second)

	// 5xx should still be returned (not swallowed) while breaker is closed.
	req, _ := http.NewRequest("GET", upstream.URL, nil)
	resp, err := bt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}
