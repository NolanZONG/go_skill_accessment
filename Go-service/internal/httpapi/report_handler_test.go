package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/school-mgmt/go-service/internal/backend"
	"github.com/school-mgmt/go-service/internal/service"
)

type stubSvc struct {
	res service.Result
	err error
}

func (s stubSvc) GenerateReport(_ context.Context, _ int) (service.Result, error) {
	return s.res, s.err
}

func newTestRouter(svc ReportGenerator) http.Handler {
	return NewRouter(svc, slog.Default())
}

func TestReportHandler_Success_ReturnsPDF(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4 sample")
	svc := stubSvc{res: service.Result{StudentID: 6, Path: "/tmp/student_6.pdf", Bytes: pdfBytes}}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/students/6/report", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/pdf", res.Header.Get("Content-Type"))
	assert.Contains(t, res.Header.Get("Content-Disposition"), `student_6.pdf`)
	assert.Equal(t, "/tmp/student_6.pdf", res.Header.Get("X-Pdf-Path"))

	body, _ := io.ReadAll(res.Body)
	assert.Equal(t, pdfBytes, body)
}

func TestReportHandler_InvalidID_Returns400(t *testing.T) {
	r := newTestRouter(stubSvc{})

	for _, raw := range []string{"abc", "-1", "0"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/students/"+raw+"/report", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "id=%s", raw)
		assert.Contains(t, rec.Body.String(), "INVALID_ID")
	}
}

func TestReportHandler_ErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		status   int
		codeText string
	}{
		{"not_found", backend.ErrStudentNotFound, http.StatusNotFound, "STUDENT_NOT_FOUND"},
		{"breaker", backend.ErrBreakerOpen, http.StatusServiceUnavailable, "BACKEND_BREAKER_OPEN"},
		{"auth", backend.ErrAuthentication, http.StatusBadGateway, "BACKEND_AUTH_FAILED"},
		{"timeout", context.DeadlineExceeded, http.StatusGatewayTimeout, "BACKEND_TIMEOUT"},
		{"unavailable", backend.ErrBackendUnavailable, http.StatusBadGateway, "BACKEND_UNAVAILABLE"},
		{"unknown", errSentinel("boom"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRouter(stubSvc{err: tc.err})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/students/1/report", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tc.status, rec.Code)
			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tc.codeText, body["code"])
			assert.NotEmpty(t, body["message"])
		})
	}
}

func TestRouter_HealthCheck(t *testing.T) {
	r := newTestRouter(stubSvc{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, strings.Contains(rec.Body.String(), `"ok"`))
}

func TestRouter_AttachesRequestID(t *testing.T) {
	r := newTestRouter(stubSvc{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

type errSentinel string

func (e errSentinel) Error() string { return string(e) }
