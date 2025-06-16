module example

go 1.24.2

replace github.com/go-chi/metrics => ../

// replace github.com/go-chi/httplog/v3 => ../../httplog
// replace github.com/golang-cz/devslog => ../../../golang-cz/devslog

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/go-chi/httplog/v3 v3.2.0
	github.com/go-chi/httprate v0.15.0
	github.com/go-chi/metrics v0.0.0-00010101000000-000000000000
	github.com/golang-cz/devslog v0.0.15
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	golang.org/x/sys v0.30.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)
