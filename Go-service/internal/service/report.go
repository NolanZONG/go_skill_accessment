// Orchestrate the report-generation flow: 
// fetch student from backend, render PDF, persist atomically. 
// Concurrent calls for the same id are coalesced via singleflight so only one fetch+render runs at a time.
package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/school-mgmt/go-service/internal/backend"
	"github.com/school-mgmt/go-service/internal/logger"
	"github.com/school-mgmt/go-service/internal/pdf"
)

// Result captures everything a caller might want to do with the generated report: 
// serve the bytes over HTTP or just inspect the file on disk.
type Result struct {
	StudentID int
	Path      string
	Bytes     []byte
}

// ReportService is the entry point exercised by the HTTP handler.
type ReportService struct {
	client         backend.StudentClient
	generator      pdf.Generator
	outDir         string
	requestTimeout time.Duration
	group          singleflight.Group
}

// Config groups the runtime knobs ReportService needs at construction.
type Config struct {
	OutputDir      string
	RequestTimeout time.Duration
}

// build a ReportService. 
// The output directory is created lazily on the first write rather than at construction time 
// so unit tests can pass a path that does not yet exist.
func New(client backend.StudentClient, gen pdf.Generator, cfg Config) *ReportService {
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 10 * time.Second
	}
	return &ReportService{
		client:         client,
		generator:      gen,
		outDir:         cfg.OutputDir,
		requestTimeout: cfg.RequestTimeout,
	}
}

// Fetche the student, renders a PDF, writes it atomically to the output directory and returns the bytes for the HTTP response. 
// Concurrent calls for the same id share a single underlying execution.
func (s *ReportService) GenerateReport(ctx context.Context, id int) (Result, error) {
	log := logger.FromContext(ctx).With("student_id", id)

	key := strconv.Itoa(id)
	v, err, shared := s.group.Do(key, func() (any, error) {
		// Detach from the caller's ctx so a single early cancellation does
		// not abort the inflight render for the other coalesced callers.
		// We still bound the work with our own timeout derived from config.
		workCtx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
		defer cancel()

		student, err := s.client.GetStudent(workCtx, id)
		if err != nil {
			return Result{}, err
		}

		bytes, err := s.generator.Render(student)
		if err != nil {
			return Result{}, fmt.Errorf("render pdf: %w", err)
		}

		path, err := s.writeAtomic(id, bytes)
		if err != nil {
			return Result{}, fmt.Errorf("persist pdf: %w", err)
		}

		log.Info("report_generated", "path", path, "size_bytes", len(bytes))
		return Result{StudentID: id, Path: path, Bytes: bytes}, nil
	})

	if shared {
		log.Debug("singleflight_shared_result")
	}

	if err != nil {
		return Result{}, err
	}
	return v.(Result), nil
}

// Write data to outDir/student_{id}.pdf using a temp-file + rename dance 
// so partial writes are never observable to other readers.
func (s *ReportService) writeAtomic(id int, data []byte) (string, error) {
	if err := os.MkdirAll(s.outDir, 0o755); err != nil {
		return "", err
	}
	finalPath := filepath.Join(s.outDir, fmt.Sprintf("student_%d.pdf", id))
	tmpPath := filepath.Join(s.outDir, fmt.Sprintf(".student_%d.pdf.tmp", id))

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		// Best-effort cleanup of the orphan tmp file on rename failure.
		_ = os.Remove(tmpPath)
		return "", err
	}
	return finalPath, nil
}
