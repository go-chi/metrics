package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an HTTP handler that serves Prometheus metrics in the standard exposition format.
// This handler should typically be mounted at the "/metrics" endpoint to follow Prometheus conventions.
// It exposes all registered metrics from the default Prometheus registry.
func Handler() http.Handler {
	return promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})
}
