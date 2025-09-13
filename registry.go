package metrics

import "github.com/prometheus/client_golang/prometheus"

// registerMetrics registers all metrics with the provided Prometheus registry.
func registerMetrics(registry prometheus.Registerer) {
	if registry == nil {
		registry = prometheus.DefaultRegisterer.(*prometheus.Registry)
	}
	registry.MustRegister(requestsCounter.vec)
	registry.MustRegister(inflightGauge.vec)
	registry.MustRegister(requestsHistogram.vec)
}
