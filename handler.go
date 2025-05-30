package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an HTTP handler that serves OpenMetrics (Prometheus metrics)
// from the default Prometheus registry in the standard exposition format.
//
// This handler should typically be mounted at the "/metrics" path and protected
// from public access, e.g. via github.com/go-chi/chi/middleware.BasicAuth middleware.
func Handler() http.Handler {
	return promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})
}
