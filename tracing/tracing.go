package tracing

import (
	"context"
	"github.com/harnash/go-middlewares"
	"net/http"
	"reflect"
	"runtime"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
)

type key int

const traceIDKey key = 911

type tracingOptions struct {
	tracer        opentracing.Tracer
	baggage       map[stringBaggageName]string
	tags          map[stringTagName]string
	logs          map[stringLogName]string
	handlerPrefix string
	handlerName   string
}

// TracingOption represents a logger option.
type TracingOption func(*tracingOptions)

// WithTracer adds list of headers that should be added to the logger
func WithTracer(tracer opentracing.Tracer) TracingOption {
	return TracingOption(func(o *tracingOptions) {
		o.tracer = tracer
	})
}

// WithBaggage will set custom baggage to the span created by middleware
func WithBaggage(name stringBaggageName, value string) TracingOption {
	return TracingOption(func(o *tracingOptions) {
		o.baggage[name] = value
	})
}

// WithTags will add custom tags tp the span create by middleware
func WithTags(name stringTagName, value string) TracingOption {
	return TracingOption(func(o *tracingOptions) {
		o.tags[name] = value
	})
}

// WithLogs will add custom log to the span create by middleware
func WithLogs(name stringLogName, value string) TracingOption {
	return TracingOption(func(o *tracingOptions) {
		o.logs[name] = value
	})
}

// WithName will define a handler name (used in span operation name) for the span created by the middleware
// Default is derived from the function name if http.Handler being wrapped
func WithName(name string) TracingOption {
	return TracingOption(func(o *tracingOptions) {
		o.handlerName = name
	})
}

// WithNamePrefix will define prefix for a operation name that is being created by middleware
func WithNamePrefix(prefix string) TracingOption {
	return TracingOption(func(o *tracingOptions) {
		o.handlerPrefix = prefix
	})
}

type stringBaggageName string
type stringTagName string
type stringLogName string

// Set will set the value of a baggage in the span
func (name stringBaggageName) Set(span opentracing.Span, value string) {
	if len(value) > 0 {
		span.SetBaggageItem(string(name), value)
	}
}

// Set will set the value of a tag in the span
func (name stringTagName) Set(span opentracing.Span, value string) {
	if len(value) > 0 {
		span.SetTag(string(name), value)
	}
}

// Set will set the value of a log in the span
func (name stringLogName) Set(span opentracing.Span, value string) {
	if len(value) > 0 {
		span.LogKV(string(name), value)
	}
}

// newTracingOptions takes functional options and returns options.
func newTracingOptions(options ...TracingOption) *tracingOptions {
	opts := &tracingOptions{
		baggage: map[stringBaggageName]string{},
		tags:    map[stringTagName]string{},
		logs:    map[stringLogName]string{},
	}

	for _, o := range options {
		o(opts)
	}
	return opts
}

// Traced is a middleware that adds OpenTracing spans to the current request context and sets some sane span tags
func Traced(options ...TracingOption) middlewares.Middleware {
	fn := func(h http.Handler) http.Handler {
		o := newTracingOptions(options...)
		if len(o.handlerName) == 0 {
			o.handlerName = runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var span opentracing.Span

			spanCtx, err := o.tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))

			if err != nil {
				span = o.tracer.StartSpan(o.handlerPrefix + o.handlerName)
			} else {
				span = o.tracer.StartSpan(o.handlerPrefix+o.handlerName, ext.RPCServerOption(spanCtx))
			}

			ext.HTTPMethod.Set(span, r.Method)
			ext.HTTPUrl.Set(span, r.URL.Path)

			for key, val := range o.baggage {
				key.Set(span, val)
			}

			for key, val := range o.tags {
				key.Set(span, val)
			}

			for key, val := range o.logs {
				key.Set(span, val)
			}

			defer span.Finish()

			if sc, ok := span.Context().(jaeger.SpanContext); ok {
				r = r.WithContext(context.WithValue(r.Context(), traceIDKey, sc.TraceID()))
			}

			h.ServeHTTP(w, r.WithContext(opentracing.ContextWithSpan(r.Context(), span)))
		})
	}

	return fn
}
