package middlewares

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"net/http"
	"net/url"
	"testing"
)

func TestLoggerInContext(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := LoggerFromRequest(r)
		if assert.NotEmpty(t, logger, "could not get logger from the context") {
			assert.IsType(t, &zap.SugaredLogger{}, logger, "invalid logger type")
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := LoggerInContext()(testHandler)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid HTTP status code")
}

func TestLoggerWithCustomLogger(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := LoggerFromRequest(r)
		if assert.NotEmpty(t, logger, "could not get logger from the context") {
			assert.IsType(t, &zap.SugaredLogger{}, logger, "invalid logger type")
			logger.Debug("some_test_massage_346#@$%^")
		}

		w.WriteHeader(http.StatusOK)
	})

	logWatcher, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(logWatcher)
	customLog := LogGetter(func() (*zap.SugaredLogger, error) {
		return logger.Sugar(), nil
	})

	handler := LoggerInContext(WithLogger(customLog))(testHandler)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid HTTP status code")

	err := logger.Sync()
	assert.NoError(t, err, "error syncing logger")
	if assert.Equal(t, logs.Len(), 1, "log not emitted to a custom logger") {
		logEntry := logs.TakeAll()[0]
		assert.Equal(t, "some_test_massage_346#@$%^", logEntry.Message, "no custom log message found")
	}
}