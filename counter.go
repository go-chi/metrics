package metrics

import "github.com/prometheus/client_golang/prometheus"

func Counter(name string, help string) CounterMetric {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: sanitizeLabel(name),
		Help: help,
	}, []string{})
	prometheus.MustRegister(vec)
	return CounterMetric{vec: vec}
}

func CounterWith[T any](name string, help string) CounterMetricLabeled[T] {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: sanitizeLabel(name),
		Help: help,
	}, labelKeys[T]())
	prometheus.MustRegister(vec)
	return CounterMetricLabeled[T]{vec: vec}
}

type CounterMetric struct {
	vec *prometheus.CounterVec
}

func (c *CounterMetric) Inc() {
	c.vec.With(prometheus.Labels{}).Inc()
}

func (c *CounterMetric) Add(value float64) {
	c.vec.With(prometheus.Labels{}).Add(value)
}

type CounterMetricLabeled[T any] struct {
	vec *prometheus.CounterVec
}

func (c *CounterMetricLabeled[T]) Inc(labels T) {
	c.vec.With(structToLabels(labels)).Inc()
}

func (c *CounterMetricLabeled[T]) Add(value float64, labels T) {
	c.vec.With(structToLabels(labels)).Add(value)
}
