package metrics

import "github.com/prometheus/client_golang/prometheus"

type InfoMetric[T any] struct {
	vec *prometheus.GaugeVec
}

func (i *InfoMetric[T]) Record(labels ...T) {
	var lbls prometheus.Labels
	if len(labels) > 0 {
		lbls = structToLabels(labels[0])
	} else {
		lbls = prometheus.Labels{}
	}
	i.vec.With(lbls).Set(1)
}
