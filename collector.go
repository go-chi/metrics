package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type CollectorOpts struct {
	Host bool // Track request "host" label.
	//Proto bool // Track request "proto" label (e.g. "HTTP/1.1").
	//WS    bool // Track WebSocket upgrades via "websocket" label (e.g. "true").
}

// Collector returns HTTP middleware for tracking Prometheus metrics for incoming HTTP requests.
func Collector(opts CollectorOpts) func(next http.Handler) http.Handler {
	type totalLabels struct {
		Host     string
		Status   string
		Endpoint string
	}
	type inflightLabels struct {
		Host string
	}

	type durationLabels struct {
		Endpoint string
	}

	durationHistogram := HistogramWith[durationLabels](
		"http_request_duration_seconds",
		"Histogram of response latency (seconds) of HTTP requests.",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
	)

	totalCounter := CounterWith[totalLabels]("http_requests_total", "Total number of HTTP requests.")

	inflightGauge := GaugeWith[inflightLabels]("http_requests_inflight", "Number of HTTP requests currently in flight.")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			var host string
			if opts.Host {
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

				totalLabels := totalLabels{
					Host:   host,
					Status: strconv.Itoa(ww.Status()),
				}
				if rctx, ok := ctx.Value(chi.RouteCtxKey).(*chi.Context); ok {
					totalLabels.Endpoint = fmt.Sprintf("%s %s", r.Method, rctx.RoutePattern())
				} else {
					totalLabels.Endpoint = fmt.Sprintf("%s %s", r.Method, r.URL.Path)
				}

				durationLabels := durationLabels{
					Endpoint: totalLabels.Endpoint,
				}

				totalCounter.Inc(totalLabels)
				durationHistogram.Observe(duration, durationLabels)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
