package logger

import (
	"github.com/harnash/go-middlewares"
	"net/http"
	"time"
)

//LoggingResponseWriter is a wrapper around ResponseWriter used to capture HTTP status code of responses
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

//NewLoggingResponseWriter creates new LoggingResponseWriter instance wrapped around net/http.ResponseWriter
func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{w, http.StatusOK}
}

//WriteHeader implements net/http.ResponseWriter's WriteHeader()
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

//AccessLog is a simple access-log style logging middleware that will log all incoming request and response info
func AccessLog() middlewares.Middleware {
	fn := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := FromRequest(r)

			logger.Info("incoming request")

			wrappedWriter := NewLoggingResponseWriter(w)
			t1 := time.Now()
			h.ServeHTTP(wrappedWriter, r)
			t2 := time.Now()

			logger.With(
				"status", wrappedWriter.statusCode,
				"duration_ns", t2.Sub(t1).Nanoseconds(),
			).Info("response generated")
		})
	}

	return fn
}
