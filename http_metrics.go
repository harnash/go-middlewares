package middlewares

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

//WriteHeader will capture http status code returned/set by the http.Handler
func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

//HTTPStats holds all the metrics regarding HTTP requests
type HTTPStats struct {
	totalRequests   *prometheus.CounterVec
	duration        *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
	requestSize     *prometheus.HistogramVec
	timeToWrite     *prometheus.HistogramVec
	handlerDuration *prometheus.HistogramVec
	handlerStatuses *prometheus.CounterVec
}

//NewHTTPStats create new HTTPStats object and initializes metrics
func NewHTTPStats() HTTPStats {
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

	return HTTPStats{totalRequests: reqCounter, duration: duration, responseSize: responseSize,
		requestSize: requestSize, timeToWrite: timeToWrite, handlerDuration: handlerDuration,
		handlerStatuses: handlerStatuses}
}

// Describe implements prometheus Collector interface.
func (s *HTTPStats) Describe(in chan<- *prometheus.Desc) {
	s.duration.Describe(in)
	s.totalRequests.Describe(in)
	s.requestSize.Describe(in)
	s.responseSize.Describe(in)
	s.timeToWrite.Describe(in)
	s.handlerDuration.Describe(in)
	s.handlerStatuses.Describe(in)
}

// Collect implements prometheus Collector interface.
func (s *HTTPStats) Collect(in chan<- prometheus.Metric) {
	s.duration.Collect(in)
	s.totalRequests.Collect(in)
	s.requestSize.Collect(in)
	s.responseSize.Collect(in)
	s.timeToWrite.Collect(in)
	s.handlerDuration.Collect(in)
	s.handlerStatuses.Collect(in)
}

//InstrumentHandler will register prometheus metrics on a given http.Handler
func (s HTTPStats) InstrumentHandler(handlerName string, next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := statusRecorder{w, 200}
		next.ServeHTTP(&d, r)

		labels := prometheus.Labels{
			"method": r.Method,
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

		s.handlerStatuses.With(labels).Inc()
	})
}

//Instrument will instrument any http.HandlerFunc with custom metrics (with custom label "handler_name")
//This is useful for gathering per-handler metrics to implement Apdex-like alerting
func (s HTTPStats) Instrument(name string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := promhttp.InstrumentHandlerResponseSize(s.responseSize, h)
		wrapped = promhttp.InstrumentHandlerCounter(s.totalRequests, wrapped)
		wrapped = promhttp.InstrumentHandlerDuration(s.duration, wrapped)
		wrapped = promhttp.InstrumentHandlerDuration(s.handlerDuration.MustCurryWith(prometheus.Labels{"handler_name": name}), wrapped)
		wrapped = promhttp.InstrumentHandlerRequestSize(s.requestSize, wrapped)
		wrapped = promhttp.InstrumentHandlerTimeToWriteHeader(s.timeToWrite, wrapped)
		wrapped = s.InstrumentHandler(name, wrapped)

		wrapped.ServeHTTP(w, r)
	})
}
