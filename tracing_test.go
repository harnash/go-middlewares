package middlewares

import (
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestBasicTracing(t *testing.T) {
	httpStats := NewHTTPStats()
	tracer := opentracing.NoopTracer{}
	err := prometheus.DefaultRegisterer.Register(httpStats)
	assert.NoError(t, err, "error while registering HTTPStats collector")
	defer prometheus.DefaultRegisterer.Unregister(httpStats)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := opentracing.SpanFromContext(r.Context())
		if span != nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotImplemented)
		}
	})

	handler := Traced(WithTracer(tracer), WithName("tests"), WithNamePrefix("prefixed_"))(testHandler)

	// quite indirect tracing test (see: testHandler for details)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{})
}