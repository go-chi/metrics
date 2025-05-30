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
				proto = getProtocol(r)
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

				route := "<no-match>"
				rctx := chi.RouteContext(ctx)
				if rctx != nil && rctx.RoutePattern() != "" {
					route = rctx.RoutePattern()
				}

				labels := requestLabels{
					Host:     host,
					Status:   strconv.Itoa(ww.Status()),
					Endpoint: fmt.Sprintf("%s %s", r.Method, route),
					Proto:    proto,
				}

				requestsTotal.Inc(labels)
				requestDuration.Observe(duration, labels)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// getProtocol determines the protocol string for the request
func getProtocol(r *http.Request) string {
	var proto string

	switch r.ProtoMajor {
	case 2:
		proto = "HTTP/2"
	case 3:
		proto = "HTTP/3"
	default:
		proto = fmt.Sprintf("HTTP/%d.%d", r.ProtoMajor, r.ProtoMinor)
	}

	// Check for WebSocket upgrade
	if isWebSocketUpgrade(r) {
		proto += " WebSocket"
	}

	return proto
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade request
func isWebSocketUpgrade(r *http.Request) bool {
	connection := strings.ToLower(r.Header.Get("Connection"))
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))

	return strings.Contains(connection, "upgrade") && upgrade == "websocket"
}
