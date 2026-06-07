// Command go-service is the entry point for the Go PDF microservice. It wires
// configuration, logger, backend client, PDF generator, report service and
// HTTP router together, then handles graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sony/gobreaker"

	"github.com/school-mgmt/go-service/internal/backend"
	"github.com/school-mgmt/go-service/internal/config"
	"github.com/school-mgmt/go-service/internal/httpapi"
	"github.com/school-mgmt/go-service/internal/logger"
	"github.com/school-mgmt/go-service/internal/pdf"
	"github.com/school-mgmt/go-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		// Fatal log + non-zero exit so container orchestrator notices.
		log := logger.New("error")
		log.Error("startup_failed", "err", err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.LogLevel)
	log.Info("config_loaded",
		"port", cfg.Port,
		"backend_base_url", cfg.BackendBaseURL,
		"output_dir", cfg.OutputDir,
		"request_timeout", cfg.RequestTimeout.String(),
	)

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return err
	}

	backendClient := backend.NewHTTPClient(backend.HTTPClientConfig{
		BaseURL:        cfg.BackendBaseURL,
		Username:       cfg.BackendUsername,
		Password:       cfg.BackendPassword,
		RequestTimeout: cfg.RequestTimeout,
		LoginTimeout:   cfg.LoginTimeout,
		BreakerSettings: gobreaker.Settings{
			Name:        "backend",
			MaxRequests: cfg.BreakerMaxRequests,
			Interval:    cfg.BreakerInterval,
			Timeout:     cfg.BreakerTimeout,
			ReadyToTrip: func(c gobreaker.Counts) bool {
				if c.Requests < cfg.BreakerMinRequests {
					return false
				}
				failureRatio := float64(c.TotalFailures) / float64(c.Requests)
				return failureRatio >= cfg.BreakerFailRatio
			},
		},
	}, log)

	gen := pdf.NewMarotoGenerator()
	reportSvc := service.New(backendClient, gen, service.Config{
		OutputDir:      cfg.OutputDir,
		RequestTimeout: cfg.RequestTimeout + cfg.LoginTimeout + 2*time.Second,
	})

	router := httpapi.NewRouter(reportSvc, log)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		log.Info("http_server_starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrCh:
		if err != nil {
			return err
		}
	case <-signalCtx.Done():
		log.Info("shutdown_signal_received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	log.Info("http_server_shutting_down", "timeout", cfg.ShutdownTimeout.String())
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful_shutdown_failed", "err", err.Error())
		return err
	}
	log.Info("http_server_stopped")
	return nil
}
