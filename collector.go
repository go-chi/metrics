package metrics

import (
	"cmp"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type CollectorOpts struct {
	HostLabel bool // Track request "host" label.
	//Proto bool // Track request "proto" label (e.g. "HTTP/1.1").
	//WS    bool // Track WebSocket upgrades via "websocket" label (e.g. "true").

	// Skip is an optional predicate function that determines whether to skip recording metrics for a given request.
	// If nil, all requests are recorded. If provided, requests where Skip returns true will not be recorded.
	Skip func(r *http.Request) bool
}

type labels struct {
	Host     string
	Status   string
	Endpoint string
}

type inflightLabels struct {
	Host string
}

// Collector returns HTTP middleware for tracking Prometheus metrics for incoming HTTP requests.
func Collector(opts CollectorOpts) func(next http.Handler) http.Handler {
	durationHistogram := HistogramWith[labels](
		"http_request_duration_seconds",
		"Histogram of response latency (seconds) of HTTP requests.",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
	)

	totalCounter := CounterWith[labels]("http_requests_total", "Total number of HTTP requests.")

	inflightGauge := GaugeWith[inflightLabels]("http_requests_inflight", "Number of HTTP requests currently in flight.")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if opts.Skip != nil && opts.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			var host string
			if opts.HostLabel {
				host = r.Host
			}

			start := time.Now()
			inflightLabels := inflightLabels{
				Host: host,
			}
			inflightGauge.Inc(inflightLabels)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				duration := time.Since(start).Seconds()
				inflightGauge.Dec(inflightLabels)

				route := cmp.Or(chi.RouteContext(ctx).RoutePattern(), "<no-match>")

				labels := labels{
					Host:     host,
					Status:   strconv.Itoa(ww.Status()),
					Endpoint: fmt.Sprintf("%s %s", r.Method, route),
				}

				totalCounter.Inc(labels)
				durationHistogram.Observe(duration, labels)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
