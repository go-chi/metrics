package metrics

import "github.com/prometheus/client_golang/prometheus"

// func Gauge(name, help string) GaugeMetric[NoLabels] {
// 	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
// 		Name: sanitizeMetricName(name),
// 		Help: help,
// 	}, []string{})
// 	prometheus.MustRegister(vec)
// 	return GaugeMetric[NoLabels]{vec: vec}
// }

func GaugeWith[T any](name, help string) GaugeMetric[T] {
	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: sanitizeLabel(name),
		Help: help,
	}, labelKeys[T]())
	prometheus.MustRegister(vec)
	return GaugeMetric[T]{vec: vec}
}

type GaugeMetric[T any] struct {
	vec *prometheus.GaugeVec
}

func (g *GaugeMetric[T]) Set(value float64, labels T) {
	g.vec.With(structToLabels(labels)).Set(value)
}

func (g *GaugeMetric[T]) Add(value float64, labels T) {
	g.vec.With(structToLabels(labels)).Add(value)
}

func (g *GaugeMetric[T]) Inc(labels T) {
	g.vec.With(structToLabels(labels)).Add(1.0)
}

func (g *GaugeMetric[T]) Dec(labels T) {
	g.vec.With(structToLabels(labels)).Add(-1.0)
}
