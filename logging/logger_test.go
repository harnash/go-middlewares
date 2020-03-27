package logging

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerInContext(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := FromRequest(r)
		if assert.NotEmpty(t, logger, "could not get logger from the context") {
			assert.IsType(t, &zap.SugaredLogger{}, logger, "invalid logger type")
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := InContext()(testHandler)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid HTTP status code")
}

func TestLoggerWithCustomLogger(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := FromRequest(r)
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

	handler := InContext(WithLogger(customLog))(testHandler)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid HTTP status code")

	err := logger.Sync()
	assert.NoError(t, err, "error syncing logger")
	if assert.Equal(t, 1, logs.Len(), "log not emitted to a custom logger") {
		logEntry := logs.TakeAll()[0]
		assert.Equal(t, "some_test_massage_346#@$%^", logEntry.Message, "no custom log message found")
	}
}

func TestLoggerWithHeaders(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := FromRequest(r)
		if assert.NotEmpty(t, logger, "could not get logger from the context") {
			assert.IsType(t, &zap.SugaredLogger{}, logger, "invalid logger type")
			logger.Debug("some_test_massage_for_header")
		}

		w.WriteHeader(http.StatusOK)
	})

	logWatcher, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(logWatcher)
	customLog := LogGetter(func() (*zap.SugaredLogger, error) {
		return logger.Sugar(), nil
	})

	handler := InContext(WithLogger(customLog), WithHeaders([]string{"goo"}))(testHandler)
	req, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err, "could not create custom request")
	w := httptest.NewRecorder()
	req.Header.Add("goo", "boo")

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "invalid http status code returned from handler")
	if assert.Equal(t, logs.Len(), 1, "log not emitted to a custom logger") {
		logEntry := logs.TakeAll()[0]
		assert.Equal(t, "some_test_massage_for_header", logEntry.Message, "no custom log message found")
		assert.Equal(t, "boo", logEntry.ContextMap()["request_header_goo"], "header not found in logger")
	}
}

func TestLoggerWithCallback(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqLogger := FromRequest(r)
		if assert.NotEmpty(t, reqLogger, "could not get logger from the context") {
			assert.IsType(t, &zap.SugaredLogger{}, reqLogger, "invalid logger type")
			reqLogger.Debug("some_test_massage_4d#@$%^")
		}

		w.WriteHeader(http.StatusOK)
	})

	logWatcher, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(logWatcher)
	callback := LogCallback(func(log *zap.SugaredLogger, r *http.Request) *zap.SugaredLogger {
		wrappedLogger := logger.With(zap.String("http_method", r.Method))
		return wrappedLogger.Sugar()
	})

	handler := InContext(WithCallback(callback))(testHandler)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid HTTP status code")

	err := logger.Sync()
	assert.NoError(t, err, "error syncing logger")
	if assert.Equal(t, 1, logs.Len(), "log not emitted to a custom logger") {
		logEntry := logs.TakeAll()[0]
		assert.Equal(t, "some_test_massage_4d#@$%^", logEntry.Message, "no log message found")
		if assert.Contains(t, logEntry.ContextMap(), "http_method", "no custom field key found") {
			assert.Equal(t, "GET", logEntry.ContextMap()["http_method"], "no custom field value found")
		}
	}
}
