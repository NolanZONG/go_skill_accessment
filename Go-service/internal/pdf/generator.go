// The Generator interface lets callers swap the concrete implementation in tests.
package pdf

import "github.com/school-mgmt/go-service/internal/backend"

// Generator turns a Student value into a self-contained PDF byte stream.
// Implementations must be safe for concurrent use because the service layer
// may render multiple students in parallel.
type Generator interface {
	Render(student *backend.Student) ([]byte, error)
}
