package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Setup initializes Prometheus metrics and returns the HTTP handler.
func Setup() http.Handler {
	return promhttp.Handler()
}
