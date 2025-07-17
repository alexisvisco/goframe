package middlewares

import (
	"net/http"
	"strings"
)

// CORSOptions configures the CORS middleware.
type CORSOptions struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CORS sets common CORS headers.
func CORS(opts *CORSOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowOrigins := []string{"*"}
			allowMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
			allowHeaders := []string{"*"}
			if opts != nil {
				if len(opts.AllowedOrigins) > 0 {
					allowOrigins = opts.AllowedOrigins
				}
				if len(opts.AllowedMethods) > 0 {
					allowMethods = opts.AllowedMethods
				}
				if len(opts.AllowedHeaders) > 0 {
					allowHeaders = opts.AllowedHeaders
				}
			}

			origin := r.Header.Get("Origin")
			if len(allowOrigins) == 1 && allowOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				for _, o := range allowOrigins {
					if o == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowHeaders, ", "))

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
