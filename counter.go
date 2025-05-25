package metrics

import "github.com/prometheus/client_golang/prometheus"

func Counter(name, help string) CounterMetric[NoLabels] {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: sanitizeLabel(name),
		Help: help,
	}, []string{})
	prometheus.MustRegister(vec)
	return CounterMetric[NoLabels]{vec: vec}
}

func CounterWith[T any](name, help string) CounterMetric[T] {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: sanitizeLabel(name),
		Help: help,
	}, labelKeys[T]())
	prometheus.MustRegister(vec)
	return CounterMetric[T]{vec: vec}
}

type CounterMetric[T any] struct {
	vec *prometheus.CounterVec
}

func (c *CounterMetric[T]) Inc(labels ...T) {
	var lbls prometheus.Labels
	if len(labels) > 0 {
		lbls = structToLabels(labels[0])
	} else {
		lbls = prometheus.Labels{}
	}
	c.vec.With(lbls).Inc()
}

func (c *CounterMetric[T]) Add(delta float64, labels ...T) {
	var lbls prometheus.Labels
	if len(labels) > 0 {
		lbls = structToLabels(labels[0])
	} else {
		lbls = prometheus.Labels{}
	}
	c.vec.With(lbls).Add(delta)
}
