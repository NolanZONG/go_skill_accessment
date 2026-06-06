// Expose the chi-based HTTP router, middleware chain, and handlers for the Go report service.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter wires the middleware chain and routes. 
// Keeping it as a single constructor makes the wiring trivially testable.
func NewRouter(svc ReportGenerator, log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(requestIDMiddleware)
	r.Use(loggerMiddleware(log))
	r.Use(recoverMiddleware)
	r.Use(accessLogMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/students/{id}/report", newReportHandler(svc).ServeHTTP)
	})

	return r
}
