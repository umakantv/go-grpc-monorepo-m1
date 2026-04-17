package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// HTTPMiddleware returns an HTTP middleware that records metrics.
// It uses raw path as-is. For pattern-based normalization, use HTTPMiddlewareWithRegistry.
func (m *Metrics) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		method := r.Method
		path := r.URL.Path

		// Track in-flight requests
		m.httpRequestsInFlight.WithLabelValues(method, path).Inc()
		defer m.httpRequestsInFlight.WithLabelValues(method, path).Dec()

		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w}

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		code := strconv.Itoa(rw.statusCode)

		m.httpRequestsTotal.WithLabelValues(method, path, code).Inc()
		m.httpRequestDuration.WithLabelValues(method, path).Observe(duration)
	})
}

// HTTPMiddlewareWithRegistry returns an HTTP middleware that records metrics
// with path normalization using a PatternRegistry.
func (m *Metrics) HTTPMiddlewareWithRegistry(registry *PatternRegistry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			method := r.Method
			path := registry.Match(r.Method, r.URL.Path)

			// Track in-flight requests
			m.httpRequestsInFlight.WithLabelValues(method, path).Inc()
			defer m.httpRequestsInFlight.WithLabelValues(method, path).Dec()

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w}

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			code := strconv.Itoa(rw.statusCode)

			m.httpRequestsTotal.WithLabelValues(method, path, code).Inc()
			m.httpRequestDuration.WithLabelValues(method, path).Observe(duration)
		})
	}
}

// HTTPMiddlewareWithPatterns returns an HTTP middleware that records metrics
// with path normalization using registered patterns from a PatternMux.
// Deprecated: Use HTTPMiddlewareWithRegistry instead.
func (m *Metrics) HTTPMiddlewareWithPatterns(mux *PatternMux) func(http.Handler) http.Handler {
	return m.HTTPMiddlewareWithRegistry(mux.Registry())
}
