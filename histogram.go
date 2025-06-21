package metrics

import "github.com/prometheus/client_golang/prometheus"

// Histogram creates a histogram metric
func Histogram(name, help string, buckets []float64) HistogramMetric {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    mustValidMetricName(name),
		Help:    help,
		Buckets: buckets,
	}, []string{})
	prometheus.MustRegister(vec)
	return HistogramMetric{vec: vec}
}

// HistogramWith creates a histogram metric with typed labels
func HistogramWith[T any](name, help string, buckets []float64) HistogramMetricLabeled[T] {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    mustValidMetricName(name),
		Help:    help,
		Buckets: buckets,
	}, getLabelKeys[T]())
	prometheus.MustRegister(vec)
	return HistogramMetricLabeled[T]{vec: vec}
}

type HistogramMetric struct {
	vec *prometheus.HistogramVec
}

func (h *HistogramMetric) Observe(value float64) {
	h.vec.With(prometheus.Labels{}).Observe(value)
}

// HistogramMetric represents a histogram metric with typed labels
type HistogramMetricLabeled[T any] struct {
	vec *prometheus.HistogramVec
}

func (h *HistogramMetricLabeled[T]) Observe(value float64, labels T) {
	h.vec.With(getLabelValues(labels)).Observe(value)
}
