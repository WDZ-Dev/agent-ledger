package otel

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestNewMetrics(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	defer func() { _ = provider.Shutdown(context.Background()) }()

	meter := provider.Meter("test")
	m, err := NewMetrics(meter)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}

	// Should not panic when recording.
	m.RecordRequest("openai", "gpt-4o-mini", 200, 150.0, 100, 50, 0.001, false, "hash123")
	m.RecordRequest("anthropic", "claude-sonnet-4-6", 200, 300.0, 200, 100, 0.005, true, "hash456")
	m.SessionStarted()
	m.SessionEnded()
	m.RecordAlert("budget_exceeded")
}

func TestSetupPrometheus(t *testing.T) {
	metrics, handler, shutdown, err := SetupPrometheus()
	if err != nil {
		t.Fatalf("SetupPrometheus: %v", err)
	}
	defer shutdown()

	if metrics == nil {
		t.Error("metrics should not be nil")
	}
	if handler == nil {
		t.Error("handler should not be nil")
	}
}
