package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all OpenTelemetry instruments for the proxy.
type Metrics struct {
	requestTotal      metric.Int64Counter
	tokensTotal       metric.Int64Counter
	costUSDTotal      metric.Float64Counter
	requestDurationMS metric.Float64Histogram
	activeSessions    metric.Int64UpDownCounter
	alertsTotal       metric.Int64Counter
	rateLimitedTotal  metric.Int64Counter
}

// NewMetrics registers all OTel instruments on the given meter.
func NewMetrics(m metric.Meter) (*Metrics, error) {
	requestTotal, err := m.Int64Counter("agentledger_request_total",
		metric.WithDescription("Total number of proxied requests"))
	if err != nil {
		return nil, err
	}

	tokensTotal, err := m.Int64Counter("agentledger_tokens_total",
		metric.WithDescription("Total tokens processed"))
	if err != nil {
		return nil, err
	}

	costUSDTotal, err := m.Float64Counter("agentledger_cost_usd_total",
		metric.WithDescription("Total cost in USD"))
	if err != nil {
		return nil, err
	}

	requestDurationMS, err := m.Float64Histogram("agentledger_request_duration_ms",
		metric.WithDescription("Request duration in milliseconds"),
		metric.WithExplicitBucketBoundaries(10, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000))
	if err != nil {
		return nil, err
	}

	activeSessions, err := m.Int64UpDownCounter("agentledger_active_sessions",
		metric.WithDescription("Number of active agent sessions"))
	if err != nil {
		return nil, err
	}

	alertsTotal, err := m.Int64Counter("agentledger_alerts_total",
		metric.WithDescription("Total alerts emitted"))
	if err != nil {
		return nil, err
	}

	rateLimitedTotal, err := m.Int64Counter("agentledger_rate_limited_total",
		metric.WithDescription("Total rate-limited requests"))
	if err != nil {
		return nil, err
	}

	return &Metrics{
		requestTotal:      requestTotal,
		tokensTotal:       tokensTotal,
		costUSDTotal:      costUSDTotal,
		requestDurationMS: requestDurationMS,
		activeSessions:    activeSessions,
		alertsTotal:       alertsTotal,
		rateLimitedTotal:  rateLimitedTotal,
	}, nil
}

// RecordRequest updates all request-related metrics in a single call.
func (m *Metrics) RecordRequest(provider, model string, statusCode int, durationMS float64,
	inputTokens, outputTokens int, costUSD float64, streaming bool, apiKeyHash string) {

	ctx := context.Background()

	attrs := metric.WithAttributes(
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.Int("status_code", statusCode),
		attribute.Bool("streaming", streaming),
	)

	m.requestTotal.Add(ctx, 1, attrs)
	m.requestDurationMS.Record(ctx, durationMS, attrs)

	tokenAttrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("model", model),
	}
	m.tokensTotal.Add(ctx, int64(inputTokens),
		metric.WithAttributes(append(tokenAttrs, attribute.String("type", "prompt"))...))
	m.tokensTotal.Add(ctx, int64(outputTokens),
		metric.WithAttributes(append(tokenAttrs, attribute.String("type", "completion"))...))

	m.costUSDTotal.Add(ctx, costUSD, metric.WithAttributes(
		attribute.String("provider", provider),
		attribute.String("model", model),
		attribute.String("api_key_hash", apiKeyHash),
	))
}

// SessionStarted increments the active sessions gauge.
func (m *Metrics) SessionStarted() {
	m.activeSessions.Add(context.Background(), 1)
}

// SessionEnded decrements the active sessions gauge.
func (m *Metrics) SessionEnded() {
	m.activeSessions.Add(context.Background(), -1)
}

// RecordRateLimited increments the rate limited counter.
func (m *Metrics) RecordRateLimited(apiKeyHash string) {
	m.rateLimitedTotal.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String("api_key_hash", apiKeyHash),
	))
}

// RecordAlert increments the alert counter for the given type.
func (m *Metrics) RecordAlert(alertType string) {
	m.alertsTotal.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String("type", alertType),
	))
}
