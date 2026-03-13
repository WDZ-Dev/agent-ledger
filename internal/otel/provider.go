package otel

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// SetupPrometheus creates an OTel MeterProvider backed by the Prometheus
// exporter and returns the Metrics instruments and an HTTP handler for /metrics.
func SetupPrometheus() (*Metrics, http.Handler, func(), error) {
	exporter, err := promexporter.New()
	if err != nil {
		return nil, nil, nil, err
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	meter := provider.Meter("agentledger")

	metrics, err := NewMetrics(meter)
	if err != nil {
		return nil, nil, nil, err
	}

	shutdown := func() {
		_ = provider.Shutdown(context.Background())
	}

	return metrics, promhttp.Handler(), shutdown, nil
}
