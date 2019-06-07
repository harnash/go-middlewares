package recover

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestRecovery(t *testing.T) {
	recovery := NewRecover()
	err := prometheus.DefaultRegisterer.Register(recovery)
	assert.NoError(t, err, "error while registering Recover collector")
	defer prometheus.DefaultRegisterer.Unregister(recovery)

	handler := recovery.Instrument()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("doh!") }))

	// initially all http stats should be zero
	assert.HTTPBodyContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `go_panics_caught_total 0`, "go_panics_caught_total did not increment")
	// should increment after a panic
	assert.HTTPError(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid status code")
	assert.HTTPBodyContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `go_panics_caught_total 1`, "go_panics_caught_total did not increment")
}
