package metrics

import (
	"net/http"

	"github.com/harnash/go-middlewares"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type options struct {
	metrics HttpStatsMetrics
	handlerName string
}

// Option represents a logger option.
type Option func(*options)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func WithMetrics(metrics HttpStatsMetrics) Option {
	return func(o *options) {
		o.metrics = metrics
	}
}

func WithName(handlerName string) Option {
	return func(o *options) {
		o.handlerName = handlerName
	}
}

//WriteHeader will capture http status code returned/set by the http.Handler
func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

type HttpStatsMetrics interface {
	prometheus.Collector
	GetTotalRequests() *prometheus.CounterVec
	GetDuration() *prometheus.HistogramVec
	GetResponseSize() *prometheus.HistogramVec
	GetRequestSize() *prometheus.HistogramVec
	GetTimeToWrite() *prometheus.HistogramVec
	GetHandlerDuration() *prometheus.HistogramVec
	GetHandlerStatuses() *prometheus.CounterVec
}

//HTTPStats holds all the metrics regarding HTTP requests
type DefaultMetrics struct {
	totalRequests   *prometheus.CounterVec
	duration        *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
	requestSize     *prometheus.HistogramVec
	timeToWrite     *prometheus.HistogramVec
	handlerDuration *prometheus.HistogramVec
	handlerStatuses *prometheus.CounterVec
}

var defaultMetrics = newDefaultMetrics()

func (s DefaultMetrics) GetTotalRequests() *prometheus.CounterVec {
	return s.totalRequests
}

func (s DefaultMetrics) GetDuration() *prometheus.HistogramVec {
	return s.duration
}

func (s DefaultMetrics) GetResponseSize() *prometheus.HistogramVec {
	return s.responseSize
}

func (s DefaultMetrics) GetRequestSize() *prometheus.HistogramVec {
	return s.requestSize
}

func (s DefaultMetrics) GetTimeToWrite() *prometheus.HistogramVec {
	return s.timeToWrite
}

func (s DefaultMetrics) GetHandlerDuration() *prometheus.HistogramVec {
	return s.handlerDuration
}

func (s DefaultMetrics) GetHandlerStatuses() *prometheus.CounterVec {
	return s.handlerStatuses
}

// newHttpStatsOptions takes functional options and returns options.
func newHttpStatsOptions(opts ...Option) *options {
	cfg := &options{
		metrics: defaultMetrics,
		handlerName: "",
	}

	for _, o := range opts {
		o(cfg)
	}

	if cfg.handlerName == "" {
		panic("handler name is not specified - cannot gather http metrics")
	}

	return cfg
}

//newDefaultMetrics create new HTTPStats object and initializes metrics
func newDefaultMetrics() DefaultMetrics {
	reqCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "number of requests",
	}, []string{"code", "method"})

	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "duration of a requests in seconds",
	}, []string{"code", "method"})

	responseSize := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_response_size_bytes",
		Help: "size of the responses in bytes",
	}, []string{"code", "method"})

	requestSize := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_size_bytes",
		Help: "size of the requests in bytes",
	}, []string{"code", "method"})

	timeToWrite := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_time_to_write_seconds",
		Help: "tracks how long it took to write all response headers in seconds",
	}, []string{"code", "method"})

	handlerDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_handler_duration_seconds",
		Help: "track how long it took to handle request in seconds",
	}, []string{"code", "method", "handler_name"})

	handlerStatuses := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_handler_statuses_total",
		Help: "count number of responses per status bucket (2xx, 3xx, 4xx, 5xx)",
	}, []string{"method", "status_bucket", "handler_name"})

	return DefaultMetrics{totalRequests: reqCounter, duration: duration, responseSize: responseSize,
		requestSize: requestSize, timeToWrite: timeToWrite, handlerDuration: handlerDuration,
		handlerStatuses: handlerStatuses}
}

// Describe implements prometheus Collector interface.
func (s DefaultMetrics) Describe(in chan<- *prometheus.Desc) {
	s.duration.Describe(in)
	s.totalRequests.Describe(in)
	s.requestSize.Describe(in)
	s.responseSize.Describe(in)
	s.timeToWrite.Describe(in)
	s.handlerDuration.Describe(in)
	s.handlerStatuses.Describe(in)
}

// Collect implements prometheus Collector interface.
func (s DefaultMetrics) Collect(in chan<- prometheus.Metric) {
	s.duration.Collect(in)
	s.totalRequests.Collect(in)
	s.requestSize.Collect(in)
	s.responseSize.Collect(in)
	s.timeToWrite.Collect(in)
	s.handlerDuration.Collect(in)
	s.handlerStatuses.Collect(in)
}

//RegisterDefaultMetrics will register default HttpStats metrics instance in Prometheus. This is only needed if
// any handlers are instrumented with default metrics (not overridden by WithMetrics() option)
func RegisterDefaultMetrics(registerer prometheus.Registerer) error {
	return registerer.Register(defaultMetrics)
}

//UnregisterDefaultMetrics is a companion function to RegisterDefaultMetrics and must be called if RegisterDefaultMetrics
// is used to cleanup the metrics in Prometheus
func UnregisterDefaultMetrics(registerer prometheus.Registerer) {
	registerer.Unregister(defaultMetrics)
}

//Measured will instrument any http.HandlerFunc with custom metrics (with custom label "handler_name")
//This is useful for gathering per-handler metrics to implement Apdex-like alerting
func Measured(options ...Option) middlewares.Middleware {
	fn := func(h http.Handler) http.Handler {
		o := newHttpStatsOptions(options...)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := promhttp.InstrumentHandlerResponseSize(o.metrics.GetResponseSize(), h)
			wrapped = promhttp.InstrumentHandlerCounter(o.metrics.GetTotalRequests(), wrapped)
			wrapped = promhttp.InstrumentHandlerDuration(o.metrics.GetDuration(), wrapped)
			wrapped = promhttp.InstrumentHandlerDuration(o.metrics.GetHandlerDuration().MustCurryWith(prometheus.Labels{"handler_name": o.handlerName}), wrapped)
			wrapped = promhttp.InstrumentHandlerRequestSize(o.metrics.GetRequestSize(), wrapped)
			wrapped = promhttp.InstrumentHandlerTimeToWriteHeader(o.metrics.GetTimeToWrite(), wrapped)
			wrapped = instrumentPrometheus(o.handlerName, o.metrics.GetHandlerStatuses(), wrapped)

			wrapped.ServeHTTP(w, r)
		})
	}

	return fn
}

//instrumentPrometheus will register prometheus metrics on a given http.Handler
func instrumentPrometheus(handlerName string, metric *prometheus.CounterVec, next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := statusRecorder{w, 200}
		next.ServeHTTP(&d, r)

		labels := prometheus.Labels{
			"method":       r.Method,
			"handler_name": handlerName,
		}

		switch {
		case d.status >= 200 && d.status <= 299:
			labels["status_bucket"] = "2xx"
		case d.status >= 300 && d.status <= 399:
			labels["status_bucket"] = "3xx"
		case d.status >= 400 && d.status <= 499:
			labels["status_bucket"] = "4xx"
		case d.status >= 500 && d.status <= 599:
			labels["status_bucket"] = "5xx"
		default:
			labels["status_bucket"] = "unknown"
		}

		metric.With(labels).Inc()
	})
}