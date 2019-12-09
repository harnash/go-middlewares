package recover

import (
	"net/http"

	"github.com/harnash/go-middlewares"
	logger2 "github.com/harnash/go-middlewares/logger"

	"github.com/prometheus/client_golang/prometheus"
)

type options struct {
	metrics Metrics
}

//Option defines functional options interface
type Option func(*options)

//WithMetrics sets custom metric collector/container for http metrics
func WithMetrics(metrics Metrics) Option {
	return func(o *options) {
		o.metrics = metrics
	}
}

//defaultMetrics implements default metrics for panic middleware
type defaultMetrics struct {
	panicCaught prometheus.Counter
}

//Metrics is the interface that will provide collector for panic metrics
type Metrics interface {
	prometheus.Collector
	GetPanicCount() prometheus.Counter
}

var basicMetrics = newDefaultMetrics()

//GetPanicCount returns metric that tracks number of panics caught during request handling
func (d defaultMetrics) GetPanicCount() prometheus.Counter {
	return d.panicCaught
}

//newDefaultMetrics creates new Recover middleware object
func newDefaultMetrics() Metrics {
	panicsStats := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "go_panics_caught_total",
		Help: "tracks the number of panics caught by http middleware",
	})

	return defaultMetrics{panicCaught: panicsStats}
}

// Describe implements prometheus Collector interface.
func (d defaultMetrics) Describe(in chan<- *prometheus.Desc) {
	d.panicCaught.Describe(in)
}

// Collect implements prometheus Collector interface.
func (d defaultMetrics) Collect(in chan<- prometheus.Metric) {
	d.panicCaught.Collect(in)
}

//RegisterDefaultMetrics will register default HttpStats metrics instance in Prometheus. This is only needed if
// any handlers are instrumented with default metrics (not overridden by WithMetrics() option)
func RegisterDefaultMetrics(registerer prometheus.Registerer) error {
	return registerer.Register(basicMetrics)
}

//UnregisterDefaultMetrics is a companion function to RegisterDefaultMetrics and must be called if RegisterDefaultMetrics
// is used to cleanup the metrics in Prometheus
func UnregisterDefaultMetrics(registerer prometheus.Registerer) {
	registerer.Unregister(basicMetrics)
}

// newOptions takes functional options and returns options.
func newOptions(opts ...Option) *options {
	cfg := &options{
		metrics: basicMetrics,
	}

	for _, o := range opts {
		o(cfg)
	}

	return cfg
}

//PanicCatch will return an http.HandlerFunc wrapper that will catch all panics and return proper HTTP response
func PanicCatch(options ...Option) middlewares.Middleware {
	fn := func(h http.Handler) http.Handler {
		o := newOptions(options...)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := logger2.FromRequest(r)
			defer func() {
				if err := recover(); err != nil {
					o.metrics.GetPanicCount().Inc()
					if logger != nil {
						logger.With("err", err).Error("panic during request handling")
					}
					http.Error(w, "500 - Internal Server Error", http.StatusInternalServerError)
				}
			}()

			h.ServeHTTP(w, r)
		})
	}

	return fn
}
