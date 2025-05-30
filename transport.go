package metrics

import (
	"fmt"
	"net/http"
	"time"
)

var (
	clientRequestDuration = HistogramWith[outgoingRequestLabels](
		"http_client_request_duration_seconds",
		"Duration of HTTP client requests in seconds",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
	)

	clientRequestsTotal = CounterWith[outgoingRequestLabels]("http_client_requests_total", "Total number of HTTP client requests.")

	clientRequestsInflight = GaugeWith[inflightLabels]("http_client_requests_inflight", "Number of HTTP client requests currently in flight.")
)

type TransportOpts struct {
	// Host adds the request host as a "host" label to metrics.
	// WARNING: High cardinality risk - only enable for limited, known hosts.
	// Do not enable for user-input URLs, crawlers, or dynamically generated hosts.
	Host bool
}

// Transport returns a new http.RoundTripper to be used in http.Client to track metrics for outgoing HTTP requests.
// It records request duration and counts for each unique combination of HTTP method, path and status code.
// The metrics are tagged with the endpoint (method + path) and status code for detailed monitoring.
func Transport(opts TransportOpts) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
			startTime := time.Now().UTC()

			// Create labels for inflight tracking (before request starts)
			inflightLabels := inflightLabels{}
			if req.URL.Host != "" && opts.Host {
				inflightLabels.Host = req.URL.Host
			}

			// Increment inflight counter
			clientRequestsInflight.Inc(inflightLabels)

			// Defer recording metrics after the request is complete
			defer func() {
				// Decrement inflight counter
				clientRequestsInflight.Dec(inflightLabels)

				// Create labels based on enabled options
				labels := requestLabels{}

				if resp != nil {
					labels.Status = fmt.Sprintf("%d", resp.StatusCode)
				} else {
					labels.Status = "error"
				}

				if req.URL.Host != "" && opts.Host {
					labels.Host = req.URL.Host
				}

				// Record metrics
				duration := time.Since(startTime).Seconds()
				clientRequestDuration.Observe(duration, labels)
				clientRequestsTotal.Inc(labels)
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
