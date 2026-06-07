package backend

import (
	"context"
	"sync"
	"sync/atomic"
)

// MockStudentClient is a hand-rolled mock of StudentClient suitable for use in
// tests across multiple packages. Behaviour is configured via the public
// fields prior to handing it to the system under test.
type MockStudentClient struct {
	// GetStudentFn lets a test override the entire GetStudent behaviour.
	// When nil, MockStudentClient falls back to returning a stub Student
	// whose ID matches the request and Name is "stub".
	GetStudentFn func(ctx context.Context, id int) (*Student, error)

	mu    sync.Mutex
	calls int32
}

// GetStudent implements StudentClient. 
// It counts invocations so tests can assert call counts (e.g. for singleflight de-duplication).
func (m *MockStudentClient) GetStudent(ctx context.Context, id int) (*Student, error) {
	atomic.AddInt32(&m.calls, 1)
	if m.GetStudentFn != nil {
		return m.GetStudentFn(ctx, id)
	}
	return &Student{ID: id, Name: "stub"}, nil
}

// Calls reports how many times GetStudent has been invoked since the mock was constructed.
func (m *MockStudentClient) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int(atomic.LoadInt32(&m.calls))
}
