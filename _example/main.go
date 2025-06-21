package main

import (
	"context"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/go-chi/httprate"
	"github.com/go-chi/metrics"
	"github.com/golang-cz/devslog"
)

func main() {
	isProduction := os.Getenv("ENV") == "production"

	logger := slog.New(logHandler(isProduction))

	r := chi.NewRouter()
	r.Use(middleware.Heartbeat("/ping"))

	// Collect metrics for incoming HTTP requests automatically.
	r.Use(metrics.Collector(metrics.CollectorOpts{
		Host:  false,
		Proto: true,
		Skip: func(r *http.Request) bool {
			return r.Method == "OPTIONS" || r.URL.Path == "/metrics"
		},
	}))

	// Request logger
	r.Use(httplog.RequestLogger(logger, &httplog.Options{
		Level:         slog.LevelInfo,
		Schema:        httplog.SchemaECS.Concise(true),
		RecoverPanics: true,
		Skip: func(r *http.Request, httpStatus int) bool {
			return r.Method == "OPTIONS" || r.URL.Path == "/metrics"
		},
	}))

	// Rate limiter.
	r.Use(httprate.LimitByIP(100, time.Second))

	r.Handle("/metrics", metrics.Handler())
	r.Get("/fast", handleFast)
	r.Get("/slow", handleSlow)
	r.Get("/error", handleError)
	r.With(middleware.Timeout(1*time.Second)).Get("/timeout", handleSlow)

	// Simulate traffic and collect metrics for outgoing HTTP requests.
	{
		// Collect metrics for outgoing HTTP requests automatically.
		transport := metrics.Transport(metrics.TransportOpts{
			Host: true,
		})

		// NOTE: Check out https://github.com/go-chi/transport for better transport chaining.
		http.DefaultClient.Transport = transport(http.DefaultTransport)

		go simulateTraffic()
	}

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
	select {
	case <-time.After(5 * time.Second):
		w.Write([]byte("This was a slow operation!\n"))
		return
	case <-r.Context().Done():
		// Disconnected or timeout.
	}
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

var jobCounter = metrics.CounterWith[*jobLabels]("jobs_processed_total", "Number of jobs processed")

func doWork(dur time.Duration) {
	time.Sleep(dur)

	if rand.Intn(100) > 90 {
		jobCounter.Inc(&jobLabels{Name: "job", Status: "success"})
	} else {
		jobCounter.Inc(&jobLabels{Name: "job", Status: "error"})
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
		endpoints := []string{
			"http://localhost:8022/fast",
			"http://localhost:8022/slow",
			"http://localhost:8022/error",
			"http://localhost:8022/timeout",
			"http://localhost:8022/not-found",
		}

		for {
			// Pick a random endpoint
			endpoint := endpoints[rand.Intn(len(endpoints))]

			// Simulate random client disconnects.
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rand.Intn(7000)+1000)*time.Millisecond)
			defer cancel()

			// Make the request.
			req, _ := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
			}

			// Wait random amount of time between 1ms..100ms
			time.Sleep(time.Duration(rand.Intn(100)+1) * time.Millisecond)
		}
	}()
}

func logHandler(isProduction bool) slog.Handler {
	handlerOpts := &slog.HandlerOptions{
		AddSource:   true,
		ReplaceAttr: httplog.SchemaECS.Concise(true).ReplaceAttr,
	}

	if isProduction {
		// JSON logs for production.
		return slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	// Pretty logs for localhost development.
	return devslog.NewHandler(os.Stdout, &devslog.Options{
		SortKeys:           true,
		MaxErrorStackTrace: 5,
		MaxSlicePrintSize:  20,
		HandlerOptions:     handlerOpts,
	})
}
