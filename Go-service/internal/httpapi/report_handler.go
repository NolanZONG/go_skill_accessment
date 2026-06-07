package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/school-mgmt/go-service/internal/backend"
	"github.com/school-mgmt/go-service/internal/logger"
	"github.com/school-mgmt/go-service/internal/service"
)

type ReportGenerator interface {
	GenerateReport(ctx context.Context, id int) (service.Result, error)
}

type reportHandler struct {
	svc ReportGenerator
}

func newReportHandler(svc ReportGenerator) *reportHandler {
	return &reportHandler{svc: svc}
}

// Implement GET /api/v1/students/{id}/report. 
// On success it streams the PDF body
// on failure it maps domain sentinel errors to HTTP status codes and a JSON error body.
func (h *reportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	rawID := chi.URLParam(r, "id")
	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "INVALID_ID",
			fmt.Sprintf("path id %q must be a positive integer", rawID))
		return
	}

	result, err := h.svc.GenerateReport(r.Context(), id)
	if err != nil {
		mapServiceError(w, log, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", strconv.Itoa(len(result.Bytes)))
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="student_%d.pdf"`, id))
	w.Header().Set("X-Pdf-Path", result.Path)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(result.Bytes); err != nil {
		log.Warn("write_response_failed", "err", err.Error())
	}
}

func mapServiceError(w http.ResponseWriter, log interface {
	Warn(string, ...any)
	Error(string, ...any)
}, err error) {
	switch {
	case errors.Is(err, backend.ErrStudentNotFound):
		log.Warn("student_not_found", "err", err.Error())
		writeError(w, http.StatusNotFound, "STUDENT_NOT_FOUND", err.Error())

	case errors.Is(err, backend.ErrBreakerOpen):
		log.Warn("circuit_breaker_open", "err", err.Error())
		writeError(w, http.StatusServiceUnavailable, "BACKEND_BREAKER_OPEN",
			"backend is temporarily unavailable; please retry later")

	case errors.Is(err, backend.ErrAuthentication):
		log.Error("backend_auth_failed", "err", err.Error())
		writeError(w, http.StatusBadGateway, "BACKEND_AUTH_FAILED",
			"upstream authentication failed")

	case errors.Is(err, context.DeadlineExceeded):
		log.Warn("backend_timeout", "err", err.Error())
		writeError(w, http.StatusGatewayTimeout, "BACKEND_TIMEOUT",
			"upstream did not respond in time")

	case errors.Is(err, backend.ErrBackendUnavailable):
		log.Error("backend_unavailable", "err", err.Error())
		writeError(w, http.StatusBadGateway, "BACKEND_UNAVAILABLE",
			"failed to reach the upstream backend service")

	default:
		log.Error("internal_error", "err", err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"an unexpected error occurred")
	}
}
