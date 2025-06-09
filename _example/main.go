package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/go-chi/metrics"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Heartbeat("/ping"))

	// Collect metrics for incoming HTTP requests automatically.
	r.Use(metrics.Collector(metrics.CollectorOpts{
		Host:  false,
		Proto: true,
		Skip: func(r *http.Request) bool {
			return r.Method == "OPTIONS"
		},
	}))

	r.Use(httprate.LimitByIP(100, time.Second))

	r.Handle("/metrics", metrics.Handler())

	// Collect metrics for outgoing HTTP requests automatically.
	transport := metrics.Transport(metrics.TransportOpts{
		Host: true,
	})

	// NOTE: Check out https://github.com/go-chi/transport for better transport chaining.
	http.DefaultClient.Transport = transport(http.DefaultTransport)

	r.Get("/fast", handleFast)
	r.Get("/slow", handleSlow)
	r.Get("/error", handleError)

	go simulateTraffic()

	log.Println("Server starting on :8022")
	if err := http.ListenAndServe(":8022", r); err != nil {
		log.Fatal(err)
	}
}

func handleFast(w http.ResponseWriter, r *http.Request) {
	doWork(100 * time.Millisecond)

	w.Write([]byte("This was a fast operation!\n"))
}

func handleSlow(w http.ResponseWriter, r *http.Request) {
	doWork(5 * time.Second)

	w.Write([]byte("This was a slow operation!\n"))
}

func handleError(w http.ResponseWriter, r *http.Request) {
	statusCodes := []int{400, 401, 403, 500, 503}
	w.WriteHeader(statusCodes[rand.Intn(len(statusCodes))])
	w.Write([]byte("Bad request error!\n"))
}

// Strongly typed metric labels help maintain low data cardinality
// by enforcing consistent label names across the codebase.
type jobLabels struct {
	Name   string `label:"name"`
	Status string `label:"status"`
}

var jobCounter = metrics.CounterWith[jobLabels]("jobs_processed_total", "Number of jobs processed")

func doWork(dur time.Duration) {
	time.Sleep(dur)

	if rand.Intn(100) > 90 {
		jobCounter.Inc(jobLabels{Name: "job", Status: "success"})
	} else {
		jobCounter.Inc(jobLabels{Name: "job", Status: "error"})
	}
}

func simulateTraffic() {
	go func() {
		for {
			resp, err := http.DefaultClient.Get("http://example.com")
			if err == nil {
				resp.Body.Close()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	go func() {
		for {
			resp, err := http.DefaultClient.Get("http://cant-resolve-this-thing.com")
			if err == nil {
				resp.Body.Close()
			}
			time.Sleep(1500 * time.Millisecond)
		}
	}()

	go func() {
		for {
			resp, err := http.DefaultClient.Get("http://localhost:8022/health")
			if err == nil {
				resp.Body.Close()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	go func() {
		endpoints := []string{
			"http://localhost:8022/slow",
			"http://localhost:8022/fast",
			"http://localhost:8022/error",
		}

		for {
			// Pick a random endpoint
			endpoint := endpoints[rand.Intn(len(endpoints))]

			// Make the request
			resp, err := http.DefaultClient.Get(endpoint)
			if err != nil {
				log.Printf("Request to %s failed: %v", endpoint, err)
			} else {
				log.Printf("Request to %s completed with status: %d", endpoint, resp.StatusCode)
				resp.Body.Close()
			}

			// Wait random amount of time between 1ms..100ms
			sleepTime := time.Duration(rand.Intn(100)+1) * time.Millisecond
			time.Sleep(sleepTime)
		}
	}()
}
