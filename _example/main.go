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
	"github.com/go-chi/transport"
)

// Custom label types for typed metrics
type HTTPLabels struct {
	Method string
	Status string
}

type JobLabels struct {
	Name   string `label:"name"`
	Status string `label:"status"`
}

func main() {
	jobCounter := metrics.CounterWith[JobLabels]("jobs_processed_total", "Number of jobs processed")

	r := chi.NewRouter()
	r.Use(metrics.Collector(metrics.CollectorOpts{
		HostLabel: false,
		Skip: func(r *http.Request) bool {
			if r.Method == "OPTIONS" {
				return true
			}
			return false
		},
	}))

	r.Use(middleware.Heartbeat("/health"))

	r.Use(httprate.LimitByIP(100, time.Second))

	r.Handle("/metrics", metrics.Handler())

	r.Get("/slow", func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow operation
		time.Sleep(time.Duration(800+time.Now().Minute()) * time.Millisecond)

		// Record job metrics
		jobCounter.Inc(JobLabels{Name: "report", Status: "ok"})

		w.Write([]byte("This was a slow operation!\n"))
	})

	r.Get("/fast", func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow operation
		time.Sleep(time.Duration(100+time.Now().Minute()) * time.Millisecond)

		// Record job metrics
		jobCounter.Inc(JobLabels{Name: "report", Status: "ok"})

		w.Write([]byte("This was a slow operation!\n"))
	})

	r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)

		jobCounter.Inc(JobLabels{Name: "report", Status: "error"})

		statusCodes := []int{400, 401, 403, 500, 503}
		w.WriteHeader(statusCodes[rand.Intn(len(statusCodes))])
		w.Write([]byte("Bad request error!\n"))
	})

	go simulateTraffic()

	log.Println("Server starting on :8022")
	if err := http.ListenAndServe(":8022", r); err != nil {
		log.Fatal(err)
	}
}

func simulateTraffic() {
	client := &http.Client{
		Transport: transport.Chain(http.DefaultTransport,
			metrics.Transport(metrics.TransportOpts{
				Host: true,
			}),
		),
		Timeout: 10 * time.Second,
	}

	go func() {
		for {
			resp, err := client.Get("http://example.com")
			if err == nil {
				resp.Body.Close()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	go func() {
		for {
			resp, err := client.Get("http://cant-resolve-this-thing.com")
			if err == nil {
				resp.Body.Close()
			}
			time.Sleep(1500 * time.Millisecond)
		}
	}()

	go func() {
		for {
			resp, err := client.Get("http://localhost:8022/health")
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
			resp, err := client.Get(endpoint)
			if err != nil {
				log.Printf("Request to %s failed: %v", endpoint, err)
			} else {
				log.Printf("Request to %s completed with status: %d", endpoint, resp.StatusCode)
				resp.Body.Close()
			}

			// Wait a random amount of time between 1ms..100ms
			sleepTime := time.Duration(rand.Intn(100)+1) * time.Millisecond
			time.Sleep(sleepTime)
		}
	}()
}
