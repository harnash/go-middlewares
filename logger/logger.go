package logger

import (
	"context"
	"fmt"
	"github.com/harnash/go-middlewares"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type key int

const loggerIDKey key = 119

// LogGetter is function that allows injecting custom logger into the middleware
type LogGetter func() (*zap.SugaredLogger, error)

type options struct {
	headers   []string
	logGetter LogGetter
}

// DefaultLogGetter defines default log getter for middleware
func DefaultLogGetter() (*zap.SugaredLogger, error) {
	logger, err := zap.NewProductionConfig().Build()
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}

// Option represents a logger option.
type Option func(*options)

// WithHeaders adds list of headers that should be added to the logger
func WithHeaders(headers []string) Option {
	return Option(func(o *options) {
		o.headers = headers
	})
}

// WithLogger adds logger getter to be stored within the request's context
func WithLogger(getter LogGetter) Option {
	return Option(func(o *options) {
		o.logGetter = getter
	})
}

// newLoggerOptions takes functional options and returns options.
func newLoggerOptions(opts ...Option) *options {
	cfg := &options{logGetter: DefaultLogGetter}

	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

// LoggerInContext is a middleware that will inject standard logger instance into the context which can be used for
// per-request logging
func LoggerInContext(options ...Option) middlewares.Middleware {
	fn := func(h http.Handler) http.Handler {
		o := newLoggerOptions(options...)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger, err := o.logGetter()
			if err != nil {
				panic(fmt.Sprintf("could not get the logger from context: %v+", err))
			}
			for _, header := range o.headers {
				headerName := http.CanonicalHeaderKey(header)
				if val := r.Header.Get(headerName); len(val) > 0 {
					logger = logger.With("request_header_"+strings.ToLower(headerName), val)
				}
			}

			h.ServeHTTP(w, r.WithContext(AddLoggerToContext(r.Context(), logger)))
		})
	}

	return fn
}

// FromRequest will return current logger embedded in the given request object
func FromRequest(r *http.Request) *zap.SugaredLogger {
	return FromContext(r.Context())
}

// FromContext will return current logger from the given context.Context object
func FromContext(ctx context.Context) *zap.SugaredLogger {
	logger := ctx.Value(loggerIDKey)
	if logger == nil {
		return nil
	}
	return logger.(*zap.SugaredLogger)
}

// AddLoggerToContext adds given logger to the context.Context and returns new context
func AddLoggerToContext(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerIDKey, logger)
}
