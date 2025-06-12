package middlewares

import (
	"net/http"
	"time"

	"github.com/alexisvisco/goframe/core/helpers/clog"
)

// MiddlewareOptions configures the HTTP middleware
type MiddlewareOptions struct{}

// LoggerMiddleware creates HTTP middleware that logs after the request is handled
func LoggerMiddleware(_ *MiddlewareOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Get or create a Line from context and attach logger
			ctx, line := clog.FromContext(r.Context())

			// Add basic request info
			line.Add("method", r.Method).
				Add("path", r.URL.Path).
				Add("remote_addr", r.RemoteAddr)

			// GenerateHandler response writer wrapper to capture status
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Execute the next handler
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Log after the handler completes
			duration := time.Since(start)
			line.Add("status_code", rw.statusCode).
				Add("duration_ms", duration.Milliseconds()).
				Log("http request")
		})
	}
}

// responseWriter captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
