package middlewares

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestBasicTracing(t *testing.T) {
	httpStats := NewHTTPStats()
	tracer := mocktracer.New()
	err := prometheus.DefaultRegisterer.Register(httpStats)
	assert.NoError(t, err, "error while registering HTTPStats collector")
	defer prometheus.DefaultRegisterer.Unregister(httpStats)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := opentracing.SpanFromContext(r.Context())
		assert.NotNil(t, span, "could not get span from the context")
		w.WriteHeader(http.StatusOK)
	})

	handler := Traced(WithTracer(tracer), WithName("tests"), WithNamePrefix("prefixed_"))(testHandler)

	// quite indirect tracing test (see: testHandler for details)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{})
	assert.Len(t, tracer.FinishedSpans(), 1, "did not register any span")
}

func TestTracingBaggage(t *testing.T) {
	httpStats := NewHTTPStats()
	tracer := mocktracer.New()
	err := prometheus.DefaultRegisterer.Register(httpStats)
	assert.NoError(t, err, "error while registering HTTPStats collector")
	defer prometheus.DefaultRegisterer.Unregister(httpStats)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := opentracing.SpanFromContext(r.Context())
		assert.NotNil(t, span, "could not get span from the context")
		w.WriteHeader(http.StatusOK)
	})

	handler := Traced(WithTracer(tracer), WithBaggage("foo", "bar"))(testHandler)

	// quite indirect tracing test (see: testHandler for details)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{})
	if assert.Len(t, tracer.FinishedSpans(), 1, "did not register any span") {
		span := tracer.FinishedSpans()[0]
		assert.Equal(t, "bar", span.BaggageItem("foo"), "invalid or missing baggage item")
	}
}

func TestTracingLogging(t *testing.T) {
	httpStats := NewHTTPStats()
	tracer := mocktracer.New()
	err := prometheus.DefaultRegisterer.Register(httpStats)
	assert.NoError(t, err, "error while registering HTTPStats collector")
	defer prometheus.DefaultRegisterer.Unregister(httpStats)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := opentracing.SpanFromContext(r.Context())
		assert.NotNil(t, span, "could not get span from the context")
		w.WriteHeader(http.StatusOK)
	})

	handler := Traced(WithTracer(tracer), WithLogs("entry", "not_empty"))(testHandler)

	// quite indirect tracing test (see: testHandler for details)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{})
	if assert.Len(t, tracer.FinishedSpans(), 1, "did not register any span") {
		span := tracer.FinishedSpans()[0]
		if assert.Len(t, span.Logs(), 1,"invalid or missing log record") {
			logKV := span.Logs()[0]
			if assert.Len(t, logKV.Fields, 1, "no log fields recorded") {
				assert.Equal(t, "not_empty", logKV.Fields[0].ValueString, "invalid log value")
				assert.Equal(t, "entry", logKV.Fields[0].Key, "invalid log key")
			}
		}
	}
}

func TestTracingTag(t *testing.T) {
	httpStats := NewHTTPStats()
	tracer := mocktracer.New()
	err := prometheus.DefaultRegisterer.Register(httpStats)
	assert.NoError(t, err, "error while registering HTTPStats collector")
	defer prometheus.DefaultRegisterer.Unregister(httpStats)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := opentracing.SpanFromContext(r.Context())
		assert.NotNil(t, span, "could not get span from the context")
		w.WriteHeader(http.StatusOK)
	})

	handler := Traced(WithTracer(tracer), WithTags("some_tag", "localtest"))(testHandler)

	// quite indirect tracing test (see: testHandler for details)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{})
	if assert.Len(t, tracer.FinishedSpans(), 1, "did not register any span") {
		span := tracer.FinishedSpans()[0]
		if assert.NotEmpty(t, span.Tags(), "invalid or missing tags") {
			tags := span.Tags()
			if assert.Contains(t, tags, "some_tag", "missing tag") {
				assert.Equal(t, "localtest", tags["some_tag"], "invalid tag value")
			}
		}
	}
}