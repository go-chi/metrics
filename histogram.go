package metrics

import "github.com/prometheus/client_golang/prometheus"

// // Histogram creates a histogram metric without labels
// func Histogram(name, help string, buckets []float64) HistogramMetric[NoLabels] {
// 	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
// 		Name:    sanitizeLabel(name),
// 		Help:    help,
// 		Buckets: buckets,
// 	}, []string{})
// 	prometheus.MustRegister(vec)
// 	return HistogramMetric[NoLabels]{vec: vec}
// }

// HistogramWith creates a histogram metric with typed labels
func HistogramWith[T any](name, help string, buckets []float64) HistogramMetric[T] {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    mustBeValidMetricName(name),
		Help:    help,
		Buckets: buckets,
	}, mustLabelKeys[T]())
	prometheus.MustRegister(vec)
	return HistogramMetric[T]{vec: vec}
}

// HistogramMetric represents a histogram metric with typed labels
type HistogramMetric[T any] struct {
	vec *prometheus.HistogramVec
}

func (h *HistogramMetric[T]) Observe(value float64, labels T) {
	h.vec.With(mustStructLabels(labels)).Observe(value)
}
