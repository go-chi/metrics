package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	requestDuration = HistogramWith[requestLabels](
		"http_request_duration_seconds",
		"Histogram of response latency (seconds) of HTTP requests.",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
	)

	requestsTotal = CounterWith[requestLabels]("http_requests_total", "Total number of HTTP requests.")

	requestsInflight = GaugeWith[inflightLabels]("http_requests_inflight", "Number of HTTP requests currently in flight.")
)

type CollectorOpts struct {
	// Host enables tracking of request "host" label.
	Host bool

	// Proto enables tracking of request "proto" label, e.g. "HTTP/2", "HTTP/1.1 WebSocket".
	Proto bool

	// Skip is an optional predicate function that determines whether to skip recording metrics for a given request.
	// If nil, all requests are recorded. If provided, requests where Skip returns true will not be recorded.
	Skip func(r *http.Request) bool
}

type requestLabels struct {
	Host     string `label:"host"`
	Status   string `label:"status"`
	Endpoint string `label:"endpoint"`
	Proto    string `label:"proto"`
}

type inflightLabels struct {
	Host  string `label:"host"`
	Proto string `label:"proto"`
}

// Collector returns HTTP middleware for tracking Prometheus metrics for incoming HTTP requests.
func Collector(opts CollectorOpts) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if opts.Skip != nil && opts.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			var host string
			if opts.Host {
				host = r.Host
			}

			var proto string
			if opts.Proto {
				proto = getProto(r)
			}

			start := time.Now()
			inflightLabels := inflightLabels{
				Host:  host,
				Proto: proto,
			}
			requestsInflight.Inc(inflightLabels)

			ww, ok := w.(middleware.WrapResponseWriter)
			if !ok {
				ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			}

			defer func() {
				duration := time.Since(start).Seconds()
				requestsInflight.Dec(inflightLabels)

				var endpoint string
				if rctx := chi.RouteContext(ctx); rctx != nil {
					if pattern := rctx.RoutePattern(); pattern != "" {
						endpoint = fmt.Sprintf("%s %s", r.Method, pattern)
					} else {
						endpoint = "<no-match>"
					}
				}

				status := strconv.Itoa(ww.Status())
				if status == "0" {
					status = "disconnected"
				}

				labels := requestLabels{
					Host:     host,
					Status:   status,
					Endpoint: endpoint,
					Proto:    proto,
				}

				requestsTotal.Inc(labels)
				requestDuration.Observe(duration, labels)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// getProto determines the protocol string for the request
func getProto(r *http.Request) string {
	if isWebSocketUpgrade(r) {
		return r.Proto + " WebSocket"
	}

	return r.Proto
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade request
func isWebSocketUpgrade(r *http.Request) bool {
	connection := strings.ToLower(r.Header.Get("Connection"))
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))

	return strings.Contains(connection, "upgrade") && upgrade == "websocket"
}
