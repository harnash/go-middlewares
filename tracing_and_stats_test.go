package middlewares

import (
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestTracingAndStats(t *testing.T) {
	httpStats := NewHTTPStats()
	tracer := opentracing.NoopTracer{}
	err := prometheus.DefaultRegisterer.Register(httpStats)
	assert.NoError(t, err, "error while registering HTTPStats collector")
	defer prometheus.DefaultRegisterer.Unregister(httpStats)

	handler := TracedWithStats("test_handler", httpStats, tracer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))

	// initially all http stats should be zero
	assert.HTTPBodyNotContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `http_requests_total{code="200",method="get"}`, "http_requests_total did not increment")
	assert.HTTPBodyNotContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `http_handler_duration_seconds_count{code="200",handler_name="test_handler",method="get"}`, "http_handler_duration_seconds_count did not increment")

	// should increment after call
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid status code")
	assert.HTTPBodyContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `http_requests_total{code="200",method="get"} 1`, "http_requests_total did not increment")
	assert.HTTPBodyContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `http_handler_duration_seconds_count{code="200",handler_name="test_handler",method="get"} 1`, "http_handler_duration_seconds_count did not increment")
}