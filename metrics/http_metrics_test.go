package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestHttpStatsTotalRequests(t *testing.T) {
	httpStats := NewHTTPStats()
	err := prometheus.DefaultRegisterer.Register(httpStats)
	assert.NoError(t, err, "error while registering HTTPStats collector")
	defer prometheus.DefaultRegisterer.Unregister(httpStats)

	promHandler := httpStats.Instrument("test_handler", promhttp.Handler())

	// initially all http stats should be zero
	assert.HTTPBodyNotContains(t, promHandler.ServeHTTP, "GET", "/", url.Values{}, `http_requests_total{code="200",method="get"}`, "http_requests_total metric is not zero")
	// now the number of requests should increment
	assert.HTTPBodyContains(t, promHandler.ServeHTTP, "GET", "/", url.Values{}, `http_requests_total{code="200",method="get"} 1`, "http_requests_total metric didn't increment properly")
}

func TestHttpStats4xx(t *testing.T) {
	httpStats := NewHTTPStats()
	err := prometheus.DefaultRegisterer.Register(httpStats)
	assert.NoError(t, err, "error while registering HTTPStats collector")
	defer prometheus.DefaultRegisterer.Unregister(httpStats)

	customHandler := httpStats.Instrument("error_handler", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))

	assert.HTTPBodyNotContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `http_requests_total{code="403",method="get"}`, "http_requests_total is not zero for 403 statuses")
	assert.HTTPError(t, customHandler.ServeHTTP, "GET", "/", url.Values{}, "custom handler did not return error status")

	metrics := assert.HTTPBody(promhttp.Handler().ServeHTTP, "GET", "/", url.Values{})
	assert.Contains(t, metrics, `http_requests_total{code="403",method="get"} 1`, "http_requests_total did not increment for 403 statuses")
	assert.Contains(t, metrics, `http_request_duration_seconds_bucket{code="403",method="get",le="1"} 1`, "http_request_duration_seconds_bucket did not increment for 403 statuses")
	assert.Contains(t, metrics, `http_request_duration_seconds_count{code="403",method="get"} 1`, "http_handler_statuses_total did not increment for 403 statuses")
	assert.Contains(t, metrics, `http_request_size_bytes_bucket{code="403",method="get",le="+Inf"} 1`, "http_request_duration_seconds_count did not increment for 403 statuses")
	assert.Contains(t, metrics, `http_request_size_bytes_count{code="403",method="get"} 1`, "http_request_size_bytes_bucket did not increment for 403 statuses")
	assert.Contains(t, metrics, `http_response_size_bytes_bucket{code="403",method="get",le="+Inf"} 1`, "http_response_size_bytes_bucket did not increment for 403 statuses")
	assert.Contains(t, metrics, `http_response_size_bytes_count{code="403",method="get"} 1`, "http_response_size_bytes_count did not increment for 403 statuses")
	assert.Contains(t, metrics, `http_time_to_write_seconds_bucket{code="403",method="get",le="1"} 1`, "http_time_to_write_seconds_bucket did not increment for 403 statuses")
	assert.Contains(t, metrics, `http_time_to_write_seconds_count{code="403",method="get"} 1`, "http_time_to_write_seconds_count did not increment for 403 statuses")
	assert.Contains(t, metrics, `http_handler_statuses_total{handler_name="error_handler",method="GET",status_bucket="4xx"} 1`, "http_handler_statuses_total did not increment for 403 statuses")
}
