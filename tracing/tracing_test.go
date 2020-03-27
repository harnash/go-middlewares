package tracing

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/harnash/go-middlewares/http_metrics"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestBasicTracing(t *testing.T) {
	tracer := mocktracer.New()
	defer tracer.Reset()
	err := http_metrics.RegisterDefaultMetrics(prometheus.DefaultRegisterer)
	assert.NoError(t, err, "error while registering http stats collector")
	defer http_metrics.UnregisterDefaultMetrics(prometheus.DefaultRegisterer)

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

func TestTracingClientHeaders(t *testing.T) {
	tracer := mocktracer.New()
	defer tracer.Reset()
	err := http_metrics.RegisterDefaultMetrics(prometheus.DefaultRegisterer)
	assert.NoError(t, err, "error while registering http stats collector")
	defer http_metrics.UnregisterDefaultMetrics(prometheus.DefaultRegisterer)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := opentracing.SpanFromContext(r.Context())
		assert.NotNil(t, span, "could not get span from the context")
		w.WriteHeader(http.StatusOK)
	})

	handler := Traced(WithTracer(tracer), WithName("test"), WithNamePrefix("handler_"))(testHandler)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := srv.Client()
	req, err := http.NewRequest("GET", srv.URL, nil)
	if assert.NoError(t, err, "could not create client request") {
		// prepare the headers
		outerSpan := tracer.StartSpan("outer_span")
		err = opentracing.GlobalTracer().Inject(
			outerSpan.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(req.Header),
		)
		if assert.NoError(t, err, "error injecting tracing headers") {
			resp, err := client.Do(req)
			outerSpan.Finish()
			if assert.NoError(t, err, "error making HTTP request") {
				assert.Equal(t, http.StatusOK, resp.StatusCode, "invalid HTTP status code")
				spans := tracer.FinishedSpans()
				if assert.Len(t, spans, 2, "not all spans were registered") {
					assert.Equal(t, "handler_test", spans[0].OperationName, "outer span not found")
					assert.Equal(t, "outer_span", spans[1].OperationName, "handler span not found")
				}
			}
		}
	}
}

func TestTracingBaggage(t *testing.T) {
	tracer := mocktracer.New()
	defer tracer.Reset()
	err := http_metrics.RegisterDefaultMetrics(prometheus.DefaultRegisterer)
	assert.NoError(t, err, "error while registering http stats collector")
	defer http_metrics.UnregisterDefaultMetrics(prometheus.DefaultRegisterer)

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
	tracer := mocktracer.New()
	defer tracer.Reset()
	err := http_metrics.RegisterDefaultMetrics(prometheus.DefaultRegisterer)
	assert.NoError(t, err, "error while registering http stats collector")
	defer http_metrics.UnregisterDefaultMetrics(prometheus.DefaultRegisterer)

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
		if assert.Len(t, span.Logs(), 1, "invalid or missing log record") {
			logKV := span.Logs()[0]
			if assert.Len(t, logKV.Fields, 1, "no log fields recorded") {
				assert.Equal(t, "not_empty", logKV.Fields[0].ValueString, "invalid log value")
				assert.Equal(t, "entry", logKV.Fields[0].Key, "invalid log key")
			}
		}
	}
}

func TestTracingTag(t *testing.T) {
	tracer := mocktracer.New()
	err := http_metrics.RegisterDefaultMetrics(prometheus.DefaultRegisterer)
	assert.NoError(t, err, "error while registering http stats collector")
	defer http_metrics.UnregisterDefaultMetrics(prometheus.DefaultRegisterer)

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
