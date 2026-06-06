// Load runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the runtime configuration of the service. 
// All values are derived from environment variables on startup. 
// Defaults are applied when an env var is missing
// Required values without a sensible default cause Load to fail.
type Config struct {
	Port               string
	BackendBaseURL     string
	BackendUsername    string
	BackendPassword    string
	RequestTimeout     time.Duration
	LoginTimeout       time.Duration
	OutputDir          string
	BreakerMaxRequests uint32
	BreakerInterval    time.Duration
	BreakerTimeout     time.Duration
	BreakerMinRequests uint32
	BreakerFailRatio   float64
	ShutdownTimeout    time.Duration
	LogLevel           string
}

// Reads configuration from the current process environment and returns a validated Config. 
// Returns an error when a required value is missing or an invalid value cannot be parsed.
func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "80"),
		BackendBaseURL:     strings.TrimRight(getEnv("BACKEND_BASE_URL", "http://backend:5007"), "/"),
		BackendUsername:    os.Getenv("BACKEND_ADMIN_USERNAME"),
		BackendPassword:    os.Getenv("BACKEND_ADMIN_PASSWORD"),
		OutputDir:          getEnv("OUTPUT_DIR", "./output"),
		LogLevel:           strings.ToLower(getEnv("LOG_LEVEL", "info")),
		BreakerMinRequests: 5,
		BreakerFailRatio:   0.6,
	}

	var err error
	if cfg.RequestTimeout, err = parseDuration("REQUEST_TIMEOUT", "5s"); err != nil {
		return nil, err
	}
	if cfg.LoginTimeout, err = parseDuration("LOGIN_TIMEOUT", "5s"); err != nil {
		return nil, err
	}
	if cfg.BreakerInterval, err = parseDuration("BREAKER_INTERVAL", "30s"); err != nil {
		return nil, err
	}
	if cfg.BreakerTimeout, err = parseDuration("BREAKER_TIMEOUT", "20s"); err != nil {
		return nil, err
	}
	if cfg.ShutdownTimeout, err = parseDuration("SHUTDOWN_TIMEOUT", "15s"); err != nil {
		return nil, err
	}

	maxReq, err := parseUint32("BREAKER_MAX_REQUESTS", "3")
	if err != nil {
		return nil, err
	}
	cfg.BreakerMaxRequests = maxReq

	if cfg.BackendUsername == "" || cfg.BackendPassword == "" {
		return nil, errors.New("BACKEND_ADMIN_USERNAME and BACKEND_ADMIN_PASSWORD are required")
	}

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return nil, fmt.Errorf("invalid PORT %q: %w", cfg.Port, err)
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func parseDuration(key, def string) (time.Duration, error) {
	raw := getEnv(key, def)
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %s=%q: %w", key, raw, err)
	}
	return d, nil
}

func parseUint32(key, def string) (uint32, error) {
	raw := getEnv(key, def)
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid uint32 %s=%q: %w", key, raw, err)
	}
	return uint32(n), nil
}
