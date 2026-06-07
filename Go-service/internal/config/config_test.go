package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultsApplied(t *testing.T) {
	t.Setenv("BACKEND_ADMIN_USERNAME", "admin@example.com")
	t.Setenv("BACKEND_ADMIN_PASSWORD", "secret")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "80", cfg.Port)
	assert.Equal(t, "http://backend:5007", cfg.BackendBaseURL)
	assert.Equal(t, 5*time.Second, cfg.RequestTimeout)
	assert.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_TrimsTrailingSlashFromBackendURL(t *testing.T) {
	t.Setenv("BACKEND_ADMIN_USERNAME", "u")
	t.Setenv("BACKEND_ADMIN_PASSWORD", "p")
	t.Setenv("BACKEND_BASE_URL", "http://backend:5007/")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "http://backend:5007", cfg.BackendBaseURL)
}

func TestLoad_RequiresCredentials(t *testing.T) {
	t.Setenv("BACKEND_ADMIN_USERNAME", "")
	t.Setenv("BACKEND_ADMIN_PASSWORD", "")

	_, err := Load()
	require.Error(t, err)
}

func TestLoad_InvalidDuration(t *testing.T) {
	t.Setenv("BACKEND_ADMIN_USERNAME", "u")
	t.Setenv("BACKEND_ADMIN_PASSWORD", "p")
	t.Setenv("REQUEST_TIMEOUT", "not-a-duration")

	_, err := Load()
	require.Error(t, err)
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("BACKEND_ADMIN_USERNAME", "u")
	t.Setenv("BACKEND_ADMIN_PASSWORD", "p")
	t.Setenv("PORT", "abc")

	_, err := Load()
	require.Error(t, err)
}
