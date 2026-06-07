package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// tokenSource owns the cached Bearer token used for calling backend API
// It is responsible for refreshing it (re-logging in) when the token is invalidated.
type tokenSource struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client

	mu    sync.RWMutex
	token string
}

func newTokenSource(baseURL, username, password string, client *http.Client) *tokenSource {
	return &tokenSource{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: client,
	}
}

// Return a cached token if available, otherwise performs a login round trip. 
// Concurrent callers serialise around login() so the backend is hit at most once per refresh.
func (t *tokenSource) Token(ctx context.Context) (string, error) {
	t.mu.RLock()
	if t.token != "" {
		tok := t.token
		t.mu.RUnlock()
		return tok, nil
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token != "" {
		return t.token, nil
	}
	tok, err := t.login(ctx)
	if err != nil {
		return "", err
	}
	t.token = tok
	return tok, nil
}

// Clear the cached token so the next Token() call triggers a new login. 
func (t *tokenSource) Invalidate() {
	t.mu.Lock()
	t.token = ""
	t.mu.Unlock()
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"accessToken"`
}

func (t *tokenSource) login(ctx context.Context) (string, error) {
	body, err := json.Marshal(loginRequest{Username: t.username, Password: t.password})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		t.baseURL+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: login: %v", ErrBackendUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("%w: login status %d body=%s", ErrAuthentication, resp.StatusCode, string(raw))
	}

	var parsed loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("%w: decode login response: %v", ErrAuthentication, err)
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("%w: empty accessToken in login response", ErrAuthentication)
	}
	return parsed.AccessToken, nil
}
