## metrics

Go package for metrics collection, HTTP middleware and handler.

[Prometheus OpenMetrics](https://github.com/prometheus/OpenMetrics/blob/main/specification/OpenMetrics.md).

Notes:
```
Suffixes for the respective types are:
Counter: '_total', '_created'
Summary: '_count', '_sum', '_created', '' (empty)
Histogram: '_count', '_sum', '_bucket', '_created'
GaugeHistogram: '_gcount', '_gsum', '_bucket'
Info: '_info'
Gauge: '' (empty)
StateSet: '' (empty)
Unknown: '' (empty)

Type
Type specifies the MetricFamily type. Valid values are "unknown", "gauge", "counter", "stateset", "info", "histogram", "gaugehistogram", and "summary".
```