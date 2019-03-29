package middlewares

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
)

//TracedWithStats is a composite middleware that adds http metrics middleware and OpenTracing in one step
func TracedWithStats(name string, stats HTTPStats, tracer opentracing.Tracer, next http.HandlerFunc) http.HandlerFunc {
	wrapped := stats.Instrument(name, next)
	wrapped = Traced(name, tracer, wrapped)

	return wrapped
}
