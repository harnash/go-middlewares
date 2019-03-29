package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type key int

const loggerIDKey key = 119

type LogGetter func() (*zap.SugaredLogger, error)

type loggerOptions struct {
	headers []string
	logGetter LogGetter
}

func DefaultLogGetter() (*zap.SugaredLogger, error) {
	logger, err := zap.NewProductionConfig().Build()
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}

// LoggerOption represents a logger option.
type LoggerOption func(*loggerOptions)

// WithHeaders adds list of headers that should be added to the logger
func WithHeaders(headers []string) LoggerOption {
	return LoggerOption(func(o *loggerOptions) {
		o.headers = headers
	})
}

// WithLogger adds logger getter to be stored within the request's context
func WithLogger(getter LogGetter) LoggerOption {
	return LoggerOption(func(o *loggerOptions) {
		o.logGetter = getter
	})
}

// newLoggerOptions takes functional options and returns options.
func newLoggerOptions(options ...LoggerOption) *loggerOptions {
	opts := &loggerOptions{logGetter:DefaultLogGetter}

	for _, o := range options {
		o(opts)
	}
	return opts
}

//LoggerInContext is a middleware that will inject standard logger instance into the context which can be used for
// per-request logging
func LoggerInContext(options ...LoggerOption) Middleware {
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
					logger = logger.With("request_header_" + strings.ToLower(headerName), val)
				}
			}

			h.ServeHTTP(w, r.WithContext(AddLoggerToContext(r.Context(), logger)))
		})
	}

	return fn
}

//LoggerFromRequest will return current logger embedded in the given request object
func LoggerFromRequest(r *http.Request) *zap.SugaredLogger {
	return LoggerFromContext(r.Context())
}

//LoggerFromContext will return current logger from the given context.Context object
func LoggerFromContext(ctx context.Context) *zap.SugaredLogger {
	return ctx.Value(loggerIDKey).(*zap.SugaredLogger)
}

//AddLoggerToContext adds given logger to the context.Context and returns new context
func AddLoggerToContext(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerIDKey, logger)
}
