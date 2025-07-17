package middlewares

import (
	"net/http"
)

// RealIPOptions configures the IP middleware.
type RealIPOptions struct {
	// Headers defines which headers are inspected to determine the client IP.
	// The first non empty header is used as RemoteAddr.
	Headers []string
}

// RealIP sets r.RemoteAddr based on the first matching header.
// Common headers like CF-Connecting-IP and X-Forwarded-For are checked by default.
func RealIP(opts *RealIPOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"}
			if opts != nil && len(opts.Headers) > 0 {
				for _, h := range opts.Headers {
					if h != "" {
						headers = append(headers, h)
					}
				}
			}
			for _, h := range headers {
				if ip := r.Header.Get(h); ip != "" {
					r.RemoteAddr = ip
					break
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
