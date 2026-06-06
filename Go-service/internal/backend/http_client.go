package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

// Captures the knobs needed to build an HTTP-backed StudentClient. 
// Defaults are applied by the caller (config package).
type HTTPClientConfig struct {
	BaseURL         string
	Username        string
	Password        string
	RequestTimeout  time.Duration
	LoginTimeout    time.Duration
	BreakerSettings gobreaker.Settings
}

// Implements StudentClient against the real backend HTTP API. 
// It owns the token cache and the circuit breaker; transient backend failures
// trip the breaker so the service short-circuits subsequent calls instead of piling up timeouts.
type HTTPClient struct {
	baseURL     string
	requestHTTP *http.Client
	tokens      *tokenSource
	breaker     *gobreaker.CircuitBreaker
	log         *slog.Logger
}

// NewHTTPClient wires an HTTPClient with the supplied configuration. 
// The returned client is safe for concurrent use.
func NewHTTPClient(cfg HTTPClientConfig, log *slog.Logger) *HTTPClient {
	if log == nil {
		log = slog.Default()
	}

	// We deliberately use two http.Client instances: the login transport has a
	// shorter, separate budget so a slow auth round trip cannot starve student
	// fetches and vice versa. Both share the same connection pool defaults.
	loginHTTP := &http.Client{Timeout: cfg.LoginTimeout}
	requestHTTP := &http.Client{Timeout: cfg.RequestTimeout}

	settings := cfg.BreakerSettings
	if settings.Name == "" {
		settings.Name = "backend"
	}
	if settings.OnStateChange == nil {
		settings.OnStateChange = func(name string, from, to gobreaker.State) {
			log.Warn("circuit_breaker_state_change",
				"name", name, "from", from.String(), "to", to.String())
		}
	}

	return &HTTPClient{
		baseURL:     cfg.BaseURL,
		requestHTTP: requestHTTP,
		tokens:      newTokenSource(cfg.BaseURL, cfg.Username, cfg.Password, loginHTTP),
		breaker:     gobreaker.NewCircuitBreaker(settings),
		log:         log,
	}
}

// GetStudent fetches the student profile, transparently handling 401-driven
// token refresh and surfacing well-known sentinel errors so handlers can map
// them to HTTP status codes.
func (c *HTTPClient) GetStudent(ctx context.Context, id int) (*Student, error) {
	result, err := c.breaker.Execute(func() (any, error) {
		return c.fetchWithRetry(ctx, id)
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, ErrBreakerOpen
		}
		return nil, err
	}
	student, ok := result.(*Student)
	if !ok {
		return nil, fmt.Errorf("%w: unexpected result type", ErrBackendUnavailable)
	}
	return student, nil
}

func (c *HTTPClient) fetchWithRetry(ctx context.Context, id int) (*Student, error) {
	student, status, err := c.fetchOnce(ctx, id)
	if err == nil {
		return student, nil
	}
	if status == http.StatusUnauthorized {
		// Token may have expired between calls; invalidate and try exactly
		// once more. We never retry past a single attempt to keep the cost
		// of an outage bounded.
		c.tokens.Invalidate()
		c.log.Info("token_invalidated_retrying", "student_id", id)
		student, _, err = c.fetchOnce(ctx, id)
	}
	return student, err
}

// fetchOnce performs a single GET /api/v1/students/{id}. Returns the parsed
// student on success and (nil, status, err) otherwise. status==0 means the
// error happened before we got an HTTP response (network, timeout, etc.).
func (c *HTTPClient) fetchOnce(ctx context.Context, id int) (*Student, int, error) {
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, 0, err
	}

	url := fmt.Sprintf("%s/api/v1/students/%d", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.requestHTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		var s Student
		if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("%w: decode: %v", ErrBackendUnavailable, err)
		}
		if s.ID == 0 {
			s.ID = id
		}
		return &s, resp.StatusCode, nil

	case resp.StatusCode == http.StatusNotFound:
		return nil, resp.StatusCode, ErrStudentNotFound

	case resp.StatusCode == http.StatusUnauthorized:
		return nil, resp.StatusCode, fmt.Errorf("%w: status 401", ErrAuthentication)

	default:
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, resp.StatusCode, fmt.Errorf("%w: status %d body=%s",
			ErrBackendUnavailable, resp.StatusCode, string(raw))
	}
}
