package middlewares

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//PromMetrics is a standard middleware that wraps http.HandlerFunc with standard prometheus metrics handler
func PromMetrics(reg prometheus.Registerer) Middleware {
	return func(h http.Handler) http.Handler {
		return promhttp.InstrumentMetricHandler(reg, h)
	}
}
