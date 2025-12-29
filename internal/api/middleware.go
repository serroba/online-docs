package api

import "net/http"

const headerUserID = "X-User-Id"

// authMiddleware extracts the user ID from the X-User-ID header
// and adds it to the request context.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(headerUserID)
		if userID == "" {
			http.Error(w, "missing X-User-ID header", http.StatusUnauthorized)

			return
		}

		ctx := withUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
