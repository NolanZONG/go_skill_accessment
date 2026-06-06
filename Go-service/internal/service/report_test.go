package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/school-mgmt/go-service/internal/backend"
)

type fakeGenerator struct {
	calls   int32
	delay   time.Duration
	payload []byte
	err     error
}

func (f *fakeGenerator) Render(_ *backend.Student) ([]byte, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.err != nil {
		return nil, f.err
	}
	if f.payload != nil {
		return f.payload, nil
	}
	return []byte("%PDF-1.4 fake"), nil
}

func newService(t *testing.T, client backend.StudentClient, gen *fakeGenerator) (*ReportService, string) {
	t.Helper()
	dir := t.TempDir()
	svc := New(client, gen, Config{OutputDir: dir, RequestTimeout: 5 * time.Second})
	return svc, dir
}

func TestReportService_GenerateReport_WritesFileAndReturnsBytes(t *testing.T) {
	mockClient := &backend.MockStudentClient{}
	gen := &fakeGenerator{}

	svc, dir := newService(t, mockClient, gen)

	res, err := svc.GenerateReport(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, 7, res.StudentID)
	assert.Equal(t, filepath.Join(dir, "student_7.pdf"), res.Path)
	assert.NotEmpty(t, res.Bytes)

	onDisk, err := os.ReadFile(res.Path)
	require.NoError(t, err)
	assert.Equal(t, res.Bytes, onDisk)
}

func TestReportService_Singleflight_CoalescesSameID(t *testing.T) {
	mockClient := &backend.MockStudentClient{
		GetStudentFn: func(_ context.Context, id int) (*backend.Student, error) {
			// Sleep just long enough for siblings to pile up on the same key.
			time.Sleep(50 * time.Millisecond)
			return &backend.Student{ID: id, Name: "Alice"}, nil
		},
	}
	gen := &fakeGenerator{delay: 20 * time.Millisecond}

	svc, _ := newService(t, mockClient, gen)

	var wg sync.WaitGroup
	const fanOut = 25
	errs := make(chan error, fanOut)
	for i := 0; i < fanOut; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.GenerateReport(context.Background(), 42)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	assert.Equal(t, 1, mockClient.Calls(),
		"singleflight should collapse same-id concurrent calls into one backend fetch")
	assert.Equal(t, int32(1), atomic.LoadInt32(&gen.calls),
		"singleflight should collapse same-id concurrent calls into one render")
}

func TestReportService_Singleflight_DoesNotCoalesceDifferentIDs(t *testing.T) {
	mockClient := &backend.MockStudentClient{
		GetStudentFn: func(_ context.Context, id int) (*backend.Student, error) {
			time.Sleep(20 * time.Millisecond)
			return &backend.Student{ID: id, Name: "x"}, nil
		},
	}
	gen := &fakeGenerator{}

	svc, _ := newService(t, mockClient, gen)

	var wg sync.WaitGroup
	const n = 10
	for i := 1; i <= n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := svc.GenerateReport(context.Background(), id)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	assert.Equal(t, n, mockClient.Calls())
	assert.Equal(t, int32(n), atomic.LoadInt32(&gen.calls))
}

func TestReportService_PropagatesClientError(t *testing.T) {
	mockClient := &backend.MockStudentClient{
		GetStudentFn: func(_ context.Context, _ int) (*backend.Student, error) {
			return nil, backend.ErrStudentNotFound
		},
	}
	gen := &fakeGenerator{}

	svc, _ := newService(t, mockClient, gen)

	_, err := svc.GenerateReport(context.Background(), 99)
	require.Error(t, err)
	assert.True(t, errors.Is(err, backend.ErrStudentNotFound))
	assert.Equal(t, int32(0), atomic.LoadInt32(&gen.calls),
		"generator should not run if the backend call fails")
}

func TestReportService_PropagatesGeneratorError(t *testing.T) {
	mockClient := &backend.MockStudentClient{}
	gen := &fakeGenerator{err: errors.New("render boom")}

	svc, _ := newService(t, mockClient, gen)

	_, err := svc.GenerateReport(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render boom")
}

func TestReportService_AtomicWrite_NoTempFileLeftOnSuccess(t *testing.T) {
	mockClient := &backend.MockStudentClient{}
	gen := &fakeGenerator{}

	svc, dir := newService(t, mockClient, gen)

	_, err := svc.GenerateReport(context.Background(), 1)
	require.NoError(t, err)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, filepath.Ext(e.Name()) == ".tmp",
			"tmp file should not be left behind: %s", e.Name())
	}
}
