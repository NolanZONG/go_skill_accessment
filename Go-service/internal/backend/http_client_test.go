package backend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeBackend lets each test wire arbitrary login and student handlers without re-writing the mux boilerplate.
type fakeBackend struct {
	loginCalls   int32
	studentCalls int32
	login        http.HandlerFunc
	student      http.HandlerFunc
}

func (f *fakeBackend) server() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.loginCalls, 1)
		f.login(w, r)
	})
	mux.HandleFunc("/api/v1/students/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.studentCalls, 1)
		f.student(w, r)
	})
	return httptest.NewServer(mux)
}

func defaultLogin(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"accessToken": "tok-123"})
}

func studentBody(id int) []byte {
	b, _ := json.Marshal(map[string]any{"id": id, "name": "Alice", "email": "alice@example.com"})
	return b
}

func newClient(t *testing.T, baseURL string) *HTTPClient {
	t.Helper()
	return NewHTTPClient(HTTPClientConfig{
		BaseURL:        baseURL,
		Username:       "admin",
		Password:       "pwd",
		RequestTimeout: 500 * time.Millisecond,
		LoginTimeout:   500 * time.Millisecond,
		BreakerSettings: gobreaker.Settings{
			Name:        "test",
			MaxRequests: 1,
			Timeout:     5 * time.Second,
			ReadyToTrip: func(c gobreaker.Counts) bool { return c.ConsecutiveFailures >= 3 },
		},
	}, nil)
}

func TestHTTPClient_GetStudent_Success(t *testing.T) {
	fb := &fakeBackend{
		login: defaultLogin,
		student: func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer tok-123", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(studentBody(42))
		},
	}
	srv := fb.server()
	defer srv.Close()

	c := newClient(t, srv.URL)
	got, err := c.GetStudent(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, 42, got.ID)
	assert.Equal(t, "Alice", got.Name)
	assert.Equal(t, int32(1), fb.loginCalls)
	assert.Equal(t, int32(1), fb.studentCalls)
}

func TestHTTPClient_GetStudent_CachesToken(t *testing.T) {
	fb := &fakeBackend{
		login: defaultLogin,
		student: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(studentBody(1))
		},
	}
	srv := fb.server()
	defer srv.Close()

	c := newClient(t, srv.URL)
	for i := 0; i < 3; i++ {
		_, err := c.GetStudent(context.Background(), i+1)
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), fb.loginCalls, "token should be cached across calls")
	assert.Equal(t, int32(3), fb.studentCalls)
}

func TestHTTPClient_NotFound(t *testing.T) {
	fb := &fakeBackend{
		login: defaultLogin,
		student: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	}
	srv := fb.server()
	defer srv.Close()

	c := newClient(t, srv.URL)
	_, err := c.GetStudent(context.Background(), 99)
	require.ErrorIs(t, err, ErrStudentNotFound)
}

func TestHTTPClient_Unauthorized_TriggersReLoginAndRetry(t *testing.T) {
	var calls int32
	fb := &fakeBackend{
		login: defaultLogin,
		student: func(w http.ResponseWriter, _ *http.Request) {
			n := atomic.AddInt32(&calls, 1)
			if n == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write(studentBody(7))
		},
	}
	srv := fb.server()
	defer srv.Close()

	c := newClient(t, srv.URL)
	got, err := c.GetStudent(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, 7, got.ID)
	assert.Equal(t, int32(2), fb.loginCalls, "second login should have happened after 401")
	assert.Equal(t, int32(2), fb.studentCalls)
}

func TestHTTPClient_ServerError_ReturnsBackendUnavailable(t *testing.T) {
	fb := &fakeBackend{
		login: defaultLogin,
		student: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "boom")
		},
	}
	srv := fb.server()
	defer srv.Close()

	c := newClient(t, srv.URL)
	_, err := c.GetStudent(context.Background(), 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackendUnavailable),
		"expected ErrBackendUnavailable, got %v", err)
}

func TestHTTPClient_Timeout_ReturnsBackendUnavailable(t *testing.T) {
	fb := &fakeBackend{
		login: defaultLogin,
		student: func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(800 * time.Millisecond)
			_, _ = w.Write(studentBody(1))
		},
	}
	srv := fb.server()
	defer srv.Close()

	c := newClient(t, srv.URL)
	_, err := c.GetStudent(context.Background(), 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackendUnavailable))
}

func TestHTTPClient_CircuitBreakerOpens(t *testing.T) {
	fb := &fakeBackend{
		login: defaultLogin,
		student: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	}
	srv := fb.server()
	defer srv.Close()

	c := newClient(t, srv.URL)
	// 3 consecutive failures trip the breaker per the settings above.
	for i := 0; i < 3; i++ {
		_, _ = c.GetStudent(context.Background(), 1)
	}
	_, err := c.GetStudent(context.Background(), 1)
	require.ErrorIs(t, err, ErrBreakerOpen)
}

func TestHTTPClient_LoginFails_ReturnsAuthenticationError(t *testing.T) {
	fb := &fakeBackend{
		login: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, "bad creds")
		},
		student: func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("student endpoint should not be reached when login fails")
		},
	}
	srv := fb.server()
	defer srv.Close()

	c := newClient(t, srv.URL)
	_, err := c.GetStudent(context.Background(), 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAuthentication) || strings.Contains(err.Error(), "login"),
		"expected auth error, got %v", err)
}
