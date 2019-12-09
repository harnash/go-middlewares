package recover

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
)

func TestRecovery(t *testing.T) {
	err := RegisterDefaultMetrics(prometheus.DefaultRegisterer)
	assert.NoError(t, err, "error while registering panic stats collector")
	defer UnregisterDefaultMetrics(prometheus.DefaultRegisterer)

	handler := PanicCatch()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("doh!") }))

	// initially all http stats should be zero
	assert.HTTPBodyContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `go_panics_caught_total 0`, "go_panics_caught_total did not increment")
	// should increment after a panic
	assert.HTTPError(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid status code")
	assert.HTTPBodyContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `go_panics_caught_total 1`, "go_panics_caught_total did not increment")
}

type testMetrics struct {
	panicMetrics prometheus.Counter
}

func (t testMetrics) Describe(in chan<- *prometheus.Desc) {
	t.panicMetrics.Describe(in)
}

func (t testMetrics) Collect(in chan<- prometheus.Metric) {
	t.panicMetrics.Collect(in)
}

func (t testMetrics) GetPanicCount() prometheus.Counter {
	return t.panicMetrics
}

func TestWithMetrics(t *testing.T) {
	metrics := testMetrics{panicMetrics:prometheus.NewCounter(prometheus.CounterOpts{
		Name: "go_test_panics_caught_total",
		Help: "tracks the number of panics caught by http middleware in tests",
	})}

	err := prometheus.DefaultRegisterer.Register(metrics)
	assert.NoError(t, err, "could not register testing metrics for recovery middleware")
	defer prometheus.DefaultRegisterer.Unregister(metrics)

	handler := PanicCatch(WithMetrics(metrics))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("doh!") }))

	// initially all http stats should be zero
	assert.HTTPBodyContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `go_test_panics_caught_total 0`, "go_test_panics_caught_total did not increment")
	// should increment after a panic
	assert.HTTPError(t, handler.ServeHTTP, "GET", "/", url.Values{}, "handler returned invalid status code")
	assert.HTTPBodyContains(t, promhttp.Handler().ServeHTTP, "GET", "/", url.Values{}, `go_test_panics_caught_total 1`, "go_test_panics_caught_total did not increment")
}
