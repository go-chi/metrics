package metrics

import "github.com/prometheus/client_golang/prometheus"

func registerMetrics(registry *prometheus.Registry) {
	if registry == nil {
		registry = prometheus.DefaultRegisterer.(*prometheus.Registry)
	}
	registry.MustRegister(requestsCounter.vec)
	registry.MustRegister(inflightGauge.vec)
	registry.MustRegister(requestsHistogram.vec)
}
