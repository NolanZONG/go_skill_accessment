// Client used to connect to the school-mgmt backend. 
// The HTTP transport details, authentication, and circuit breaker live behind the StudentClient interface
// So callers (and tests) can substitute fakes freely.
package backend

import (
	"context"
	"errors"
)

// Student mirrors the JSON shape returned by GET /api/v1/students/{id}.
// All optional fields use pointers because the upstream is permissive about missing values from the profile join.
type Student struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	Email              string  `json:"email"`
	Phone              *string `json:"phone,omitempty"`
	Gender             *string `json:"gender,omitempty"`
	DOB                *string `json:"dob,omitempty"`
	Class              *string `json:"class,omitempty"`
	Section            *string `json:"section,omitempty"`
	Roll               *int    `json:"roll,omitempty"`
	FatherName         *string `json:"fatherName,omitempty"`
	FatherPhone        *string `json:"fatherPhone,omitempty"`
	MotherName         *string `json:"motherName,omitempty"`
	MotherPhone        *string `json:"motherPhone,omitempty"`
	GuardianName       *string `json:"guardianName,omitempty"`
	GuardianPhone      *string `json:"guardianPhone,omitempty"`
	RelationOfGuardian *string `json:"relationOfGuardian,omitempty"`
	CurrentAddress     *string `json:"currentAddress,omitempty"`
	PermanentAddress   *string `json:"permanentAddress,omitempty"`
	AdmissionDate      *string `json:"admissionDate,omitempty"`
	ReporterName       *string `json:"reporterName,omitempty"`
	SystemAccess       bool    `json:"systemAccess,omitempty"`
}

// StudentClient is the surface area the service layer depends on
// the HTTP implementation is in the http_client.go.
type StudentClient interface {
	GetStudent(ctx context.Context, id int) (*Student, error)
}

// Sentinel errors callers can compare against with errors.Is to make HTTP
// status mapping in handlers explicit.
var (
	ErrStudentNotFound    = errors.New("student not found")
	ErrBackendUnavailable = errors.New("backend unavailable")
	ErrBreakerOpen        = errors.New("circuit breaker open")
	ErrAuthentication     = errors.New("backend authentication failed")
)
