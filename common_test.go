package middlewares

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testMiddleware() Middleware {
	fn := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Test-Header", "1")
			h.ServeHTTP(w, r)
		})
	}

	return fn
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestUse(t *testing.T) {
	handlerFunc := http.HandlerFunc(testHandler)

	// registering all middlewares
	h := Use(handlerFunc, testMiddleware())

	response := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "localhost/test", nil)

	assert.NoError(t, err, "could not create test Request")
	h.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "1", response.Header().Get("Test-Header"), "middleware not registered successfully")
}

func TestUseFunc(t *testing.T) {
	// registering all middlewares
	h := UseFunc(testHandler, testMiddleware())

	response := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "localhost/test", nil)

	assert.NoError(t, err, "could not create test Request")
	h.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "1", response.Header().Get("Test-Header"), "middleware not registered successfully")
}