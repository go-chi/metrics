# metrics

> Go package for metrics collection in [OpenMetrics](https://github.com/prometheus/OpenMetrics/blob/main/specification/OpenMetrics.md) format.

[![Go Reference](https://pkg.go.dev/badge/github.com/go-chi/metrics.svg)](https://pkg.go.dev/github.com/go-chi/metrics)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-chi/metrics)](https://goreportcard.com/report/github.com/go-chi/httplog)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Features

- **🚀 High Performance**: Built on top of [Prometheus](https://github.com/prometheus/client_golang) Go client with minimal overhead.
- **🌐 HTTP Middleware**: Real-time monitoring of incoming requests.
- **🔄 HTTP Transport**: Client instrumentation for outgoing requests.
- **🎯 Compatibility**: Compatible with [OpenMetrics 1.0](https://github.com/prometheus/OpenMetrics/blob/main/specification/OpenMetrics.md) collectors, e.g. Prometheus.
- **🔒 Type Safety**: Compile-time type-safe metric labels with struct tags validation.
- **🏷️ Data Cardinality**: The API helps you keep the metric label cardinality low.
- **📊 Complete Metrics**: Counter, Gauge, and Histogram metrics with customizable buckets.

## Usage

`go get github.com/go-chi/metrics@latest`

```go
package main

import (
	"github.com/go-chi/metrics"
)

func main() {
	r := chi.NewRouter()

	// Collect metrics for incoming HTTP requests automatically.
	r.Use(metrics.Collector(metrics.CollectorOpts{
		Host:  false,
		Proto: true,
		Skip: func(r *http.Request) bool {
			return r.Method != "OPTIONS"
		},
	}))

	r.Handle("/metrics", metrics.Handler())
	r.Post("/do-work", doWork)

	// Collect metrics for outgoing HTTP requests automatically.
	transport := metrics.Transport(metrics.TransportOpts{
		Host: true,
	})
	http.DefaultClient.Transport = transport(http.DefaultTransport)

	go simulateTraffic()

	log.Println("Server starting on :8022")
	if err := http.ListenAndServe(":8022", r); err != nil {
		log.Fatal(err)
	}
}

// Strongly typed metric labels help maintain low data cardinality
// by enforcing consistent label names across the codebase.
type jobLabels struct {
	Name   string `label:"name"`
	Status string `label:"status"`
}

var jobCounter = metrics.CounterWith[jobLabels]("jobs_processed_total", "Number of jobs processed")

func doWork(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second) // simulate work

	if rand.Intn(100) > 90 { // simulate error
		jobCounter.Inc(jobLabels{Name: "job", Status: "error"})
		w.Write([]byte("Job failed.\n"))
		return
	}

	jobCounter.Inc(jobLabels{Name: "job", Status: "success"})
	w.Write([]byte("Job finished successfully.\n"))
}

func simulateTraffic() {
	for {
		_, _ = client.Get("http://example.com")
		time.Sleep(500 * time.Millisecond)
	}
}
```

## Example

See [_example/main.go](./_example/main.go) and try it locally:
```sh
$ cd _example

$ go run .
```

TODO: Run Prometheus + Grafana locally.

## License
[MIT license](./LICENSE)
