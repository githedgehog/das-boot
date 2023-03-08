package seeder

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// SetHeader is a convenience handler to set a response header key/value
func AddResponseRequestID() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			if reqID != "" {
				w.Header().Set("Request-ID", reqID)
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
