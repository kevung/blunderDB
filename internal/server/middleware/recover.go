package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover converts a panic in a downstream handler into a 500 response written
// by errFn, logging the panic and its stack. It is the outermost middleware.
func Recover(logger *slog.Logger, errFn func(http.ResponseWriter, *http.Request)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if logger != nil {
						logger.Error("panic recovered",
							"panic", rec,
							"method", r.Method,
							"path", r.URL.Path,
							"stack", string(debug.Stack()),
						)
					}
					errFn(w, r)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
