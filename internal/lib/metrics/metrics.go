package metrics

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "url_shortener"
	subsystem = "http"
)

// Metrics holds all the metrics for the application
type Metrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	urlsCreatedTotal *prometheus.CounterVec
	redirectsTotal   *prometheus.CounterVec
}

// New creates a new instance of Metrics with all metrics registered
func New() *Metrics {
	m := &Metrics{
		// HTTP metrics
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed, partitioned by status code, method and path.",
			},
			[]string{"code", "method", "path"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "The HTTP request latencies in seconds.",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		urlsCreatedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "urls_created_total",
				Help:      "Total number of URLs created.",
			},
			[]string{},
		),
		redirectsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "redirects_total",
				Help:      "Total number of URL redirects.",
			},
			[]string{},
		),
	}

	return m
}

// Middleware returns a middleware that records request metrics
func (m *Metrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := NewResponseWriter(w)

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()

			path := chi.RouteContext(r.Context()).RoutePattern()
			if path == "" {
				path = r.URL.Path
			}

			m.requestDuration.WithLabelValues(r.Method, path).Observe(duration)
			m.requestsTotal.WithLabelValues(http.StatusText(ww.status), r.Method, path).Inc()
		})
	}
}

// RecordURLCreated increments the URL creation counter
func (m *Metrics) RecordURLCreated() {
	m.urlsCreatedTotal.WithLabelValues().Inc()
}

// RecordRedirect increments the redirect counter
func (m *Metrics) RecordRedirect() {
	m.redirectsTotal.WithLabelValues().Inc()
}

// Handler returns the Prometheus metrics handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

// ResponseWriter is a wrapper around http.ResponseWriter that tracks the status code
// for metrics collection
// This is needed to capture the status code written by the handler
// since the default http.ResponseWriter doesn't expose it
// and we need it for metrics collection.
type ResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{ResponseWriter: w}
}

// WriteHeader captures the status code for metrics collection
func (rw *ResponseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

// Write writes the data to the connection as part of an HTTP reply
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.status = http.StatusOK
		rw.wroteHeader = true
	}
	return rw.ResponseWriter.Write(b)
}

// Status returns the status code of the response or 0 if the response has not been written
func (rw *ResponseWriter) Status() int {
	if !rw.wroteHeader {
		return http.StatusOK
	}
	return rw.status
}

// Unwrap returns the underlying http.ResponseWriter
func (rw *ResponseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
