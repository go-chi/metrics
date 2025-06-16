package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an HTTP handler that serves Prometheus metrics from the default registry
// in the OpenMetrics exposition format.
//
// This handler should typically be mounted at "/metrics" and protected from public access,
// e.g. via middleware.BasicAuth or exposed only on a private port.
func Handler() http.Handler {
	return promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})
}
