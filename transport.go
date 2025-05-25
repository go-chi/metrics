package metrics

import (
	"fmt"
	"net/http"
	"time"
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
func Transport(baseTransport http.RoundTripper, opts TransportOpts) func(http.RoundTripper) http.RoundTripper {
	type totalLabels struct {
		Host   string
		Status string
	}
	type inflightLabels struct {
		Host string
	}

	// Create metrics for HTTP client requests
	// Use standard prometheus histogram buckets for HTTP durations (in seconds)
	durationBuckets := []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100}
	requestDuration := HistogramWith[totalLabels](
		"http_client_request_duration_seconds",
		"Duration of HTTP client requests in seconds",
		durationBuckets,
	)

	requestsTotal := CounterWith[totalLabels]("http_client_requests_total", "Total number of HTTP client requests.")

	requestsInflight := GaugeWith[inflightLabels]("http_client_requests_inflight", "Number of HTTP client requests currently in flight.")

	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripFunc(func(req *http.Request) (resp *http.Response, err error) {
			startTime := time.Now().UTC()

			// Create labels for inflight tracking (before request starts)
			inflightLabels := inflightLabels{}
			if req.URL.Host != "" && opts.Host {
				inflightLabels.Host = req.URL.Host
			}

			// Increment inflight counter
			requestsInflight.Inc(inflightLabels)

			// Defer recording metrics after the request is complete
			defer func() {
				// Decrement inflight counter
				requestsInflight.Dec(inflightLabels)

				// Create labels based on enabled options
				labels := totalLabels{}

				if resp != nil {
					// Always collect status code
					labels.Status = fmt.Sprintf("%d", resp.StatusCode)
				}

				if req.URL.Host != "" && opts.Host {
					labels.Host = req.URL.Host
				}

				// Record metrics
				duration := time.Since(startTime).Seconds()
				requestDuration.Observe(duration, labels)
				requestsTotal.Inc(labels)
			}()

			if next != nil {
				return next.RoundTrip(req)
			}
			return baseTransport.RoundTrip(req)
		})
	}
}

// RoundTripFunc, similar to http.HandlerFunc, is an adapter
// to allow the use of ordinary functions as http.RoundTrippers.
type roundTripFunc func(r *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
