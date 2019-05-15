package middlewares

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestAccessLog(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	logWatcher, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(logWatcher)
	customLog := LogGetter(func() (*zap.SugaredLogger, error) {
		return logger.Sugar(), nil
	})

	handler := AccessLog()(testHandler)
	handler = LoggerInContext(WithLogger(customLog))(handler)
	assert.HTTPSuccess(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid HTTP status code")

	err := logger.Sync()
	assert.NoError(t, err, "error syncing logger")
	if assert.Equal(t,2, logs.Len(),"log not emitted to a custom logger") {
		logEntries := logs.TakeAll()
		assert.Equal(t, "incoming request", logEntries[0].Message, "no proper access log message found")
		assert.Equal(t, "response generated", logEntries[1].Message, "no proper access log message found")
	}
}