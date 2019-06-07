package recover

import (
	"github.com/harnash/go-middlewares"
	logger2 "github.com/harnash/go-middlewares/logger"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

//Recover is a standard middleware that captures all the panics and returns 500 response to the client
type Recover struct {
	panicCaught prometheus.Counter
}

//NewRecover creates new Recover middleware object
func NewRecover() Recover {
	panicsStats := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "go_panics_caught_total",
		Help: "tracks the number of panics caught by http middleware",
	})

	return Recover{panicCaught: panicsStats}
}

// Describe implements prometheus Collector interface.
func (rc Recover) Describe(in chan<- *prometheus.Desc) {
	rc.panicCaught.Describe(in)
}

// Collect implements prometheus Collector interface.
func (rc Recover) Collect(in chan<- prometheus.Metric) {
	rc.panicCaught.Collect(in)
}

//Instrument will return an http.HandlerFunc wrapper that will catch all panics and return proper HTTP response
func (rc Recover) Instrument() middlewares.Middleware {
	fn := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := logger2.FromRequest(r)
			defer func() {
				if err := recover(); err != nil {
					rc.panicCaught.Inc()
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
