package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
)

// Recovery catches panics, logs them, and returns a structured 500 response.
func Recovery(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorContext(r.Context(), "panic recovered", slog.String("panic", fmt.Sprint(rec)))
				writeJSONError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
