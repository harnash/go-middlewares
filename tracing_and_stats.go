package middlewares

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
)

//TracedWithStats is a composite middleware that adds http metrics middleware and OpenTracing in one step
func TracedWithStats(name string, stats HTTPStats, tracer opentracing.Tracer) Middleware {
	return func(next http.Handler) http.Handler {
		wrapped := stats.Instrument(name, next)
		return Traced(WithName(name), WithTracer(tracer))(wrapped)
	}
}
