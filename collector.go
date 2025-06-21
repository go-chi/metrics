package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	requestsCounter   = CounterWith[requestLabels]("http_requests_total", "Total number of incoming HTTP requests.")
	inflightGauge     = GaugeWith[inflightLabels]("http_requests_inflight", "Number of incoming HTTP requests currently in flight.")
	requestsHistogram = HistogramWith[histogramLabels](
		"http_request_duration_seconds",
		"Response latency in seconds for completed incoming HTTP requests.",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
	)
)

// CollectorOpts configures the HTTP request metrics collector.
type CollectorOpts struct {
	// Host enables tracking of request "host" label.
	Host bool

	// Proto enables tracking of request "proto" label (e.g. "HTTP/2", "HTTP/1.1 WebSocket").
	Proto bool

	// Skip is an optional predicate function that determines whether to skip recording metrics for a given request.
	// If nil, all requests are recorded. If provided, requests where Skip returns true will not be recorded.
	Skip func(r *http.Request) bool
}

// requestLabels defines labels for the counter of total incoming HTTP requests.
type requestLabels struct {
	Host          string `label:"host"`
	Status        string `label:"status"`
	Endpoint      string `label:"endpoint"`
	Proto         string `label:"proto"`
	ClientAborted string `label:"client_aborted"`
}

// histogramLabels defines labels for the histogram of completed incoming HTTP requests.
type histogramLabels struct {
	Status   string `label:"status"`
	Endpoint string `label:"endpoint"`
}

// inflightLabels defines labels for the gauge of in-flight incoming HTTP requests.
type inflightLabels struct {
	Host  string `label:"host"`
	Proto string `label:"proto"`
}

// Collector returns HTTP middleware that automatically tracks Prometheus metrics
// for incoming HTTP requests:
// - http_requests_total: Total number of incoming HTTP requests
// - http_requests_inflight: Number of incoming HTTP requests currently in flight
// - http_request_duration_seconds: Response latency in seconds for completed requests
func Collector(opts CollectorOpts) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if opts.Skip != nil && opts.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			inflightLabels := inflightLabels{
				Host:  getHost(r, opts.Host),
				Proto: getProto(r, opts.Proto),
			}
			inflightGauge.Inc(inflightLabels)

			ww, ok := w.(middleware.WrapResponseWriter)
			if !ok {
				ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			}

			defer func() {
				duration := time.Since(start).Seconds()
				inflightGauge.Dec(inflightLabels)

				endpoint := "<no-match>"
				if rctx := chi.RouteContext(r.Context()); rctx != nil {
					if pattern := rctx.RoutePattern(); pattern != "" {
						endpoint = fmt.Sprintf("%s %s", r.Method, pattern)
					}
				}

				statusCode := ww.Status()
				if statusCode == 0 {
					// If the handler never calls w.WriteHeader(statusCode) explicitly,
					// Go's http package automatically sends HTTP 200 OK to the client.
					statusCode = 200
				}

				labels := requestLabels{
					Host:     inflightLabels.Host,
					Status:   strconv.Itoa(statusCode),
					Endpoint: endpoint,
					Proto:    inflightLabels.Proto,
				}

				if errors.Is(r.Context().Err(), context.Canceled) {
					labels.ClientAborted = "true"
				} else {
					// Observe duration of completed requests.
					requestsHistogram.Observe(duration, histogramLabels{
						Status:   labels.Status,
						Endpoint: labels.Endpoint,
					})
				}

				// Track total number of requests.
				requestsCounter.Inc(labels)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

func getHost(r *http.Request, collect bool) string {
	if !collect {
		return ""
	}
	return r.Host
}

func getProto(r *http.Request, collect bool) string {
	if !collect {
		return ""
	}
	if isWebSocketUpgrade(r) {
		return r.Proto + " WebSocket"
	}
	return r.Proto
}

func isWebSocketUpgrade(r *http.Request) bool {
	connection := strings.ToLower(r.Header.Get("Connection"))
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))

	return strings.Contains(connection, "upgrade") && upgrade == "websocket"
}
