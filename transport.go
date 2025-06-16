package metrics

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"
)

var (
	clientRequestsCounter  = CounterWith[outgoingRequestLabels]("http_client_requests_total", "Total number of outgoing HTTP requests.")
	clientInflightGauge    = GaugeWith[outgoingInflightLabels]("http_client_requests_inflight", "Number of outgoing HTTP requests currently in flight.")
	clientRequestHistogram = HistogramWith[outgoingRequestLabels](
		"http_client_request_duration_seconds",
		"Response latency in seconds for completed outgoing HTTP requests.",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
	)
)

// TransportOpts configures the HTTP client metrics transport.
type TransportOpts struct {
	// Host adds the request host as a "host" label to http_client_requests_total metric.
	// WARNING: High cardinality risk - only enable for limited, known hosts. Do not enable
	// for user-input URLs, crawlers, or dynamically generated hosts.
	Host bool
}

// outgoingRequestLabels defines labels for the counter of total outgoing HTTP requests.
type outgoingRequestLabels struct {
	Host   string `label:"host"`
	Status string `label:"status"`
}

// outgoingInflightLabels defines labels for the gauge of in-flight outgoing HTTP requests.
type outgoingInflightLabels struct {
	Host string `label:"host"`
}

// Transport returns a new http.RoundTripper that automatically tracks Prometheus metrics
// for outgoing HTTP requests:
// - http_client_requests_total: Total number of outgoing HTTP requests
// - http_client_requests_inflight: Number of outgoing HTTP requests currently in flight
// - http_client_request_duration_seconds: Response latency in seconds for completed requests
func Transport(opts TransportOpts) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
			startTime := time.Now().UTC()

			// Create labels for inflight tracking (before request starts)
			inflightLabels := outgoingInflightLabels{}
			if req.URL.Host != "" && opts.Host {
				inflightLabels.Host = req.URL.Host
			}

			// Increment inflight counter
			clientInflightGauge.Inc(inflightLabels)

			// Defer recording metrics after the request is complete
			defer func() {
				// Decrement inflight counter
				clientInflightGauge.Dec(inflightLabels)

				// Create labels based on enabled options
				labels := outgoingRequestLabels{}

				switch {
				case resp != nil:
					labels.Status = strconv.Itoa(resp.StatusCode)
				case errors.Is(err, context.DeadlineExceeded):
					labels.Status = "timeout"
				case errors.Is(err, context.Canceled):
					labels.Status = "canceled"
				default:
					labels.Status = "error"
				}

				if req.URL.Host != "" && opts.Host {
					labels.Host = req.URL.Host
				}

				// Track total number of requests.
				clientRequestsCounter.Inc(labels)

				// Observe histogram of completed requests.
				if resp != nil {
					duration := time.Since(startTime).Seconds()
					clientRequestHistogram.Observe(duration, labels)
				}
			}()

			if next != nil {
				return next.RoundTrip(req)
			}
			return http.DefaultTransport.RoundTrip(req)
		})
	}
}

// roundTripperFunc, similar to http.HandlerFunc, is an adapter
// to allow the use of ordinary functions as http.RoundTrippers.
type roundTripperFunc func(r *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
