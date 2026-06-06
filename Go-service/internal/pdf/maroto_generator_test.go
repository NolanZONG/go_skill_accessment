package pdf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/school-mgmt/go-service/internal/backend"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestMarotoGenerator_Render_ProducesPDF(t *testing.T) {
	g := NewMarotoGenerator()

	s := &backend.Student{
		ID:           42,
		Name:         "Alice Anderson",
		Email:        "alice@example.com",
		Phone:        strPtr("+1-555-0100"),
		Gender:       strPtr("Female"),
		DOB:          strPtr("2008-03-14"),
		Class:        strPtr("Grade 10"),
		Section:      strPtr("A"),
		Roll:         intPtr(7),
		FatherName:   strPtr("Bob Anderson"),
		MotherName:   strPtr("Carol Anderson"),
		SystemAccess: true,
	}

	out, err := g.Render(s)
	require.NoError(t, err)
	require.NotEmpty(t, out, "PDF bytes must not be empty")
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF-")),
		"output should start with the PDF magic header, got %q", string(out[:8]))
}

func TestMarotoGenerator_Render_RejectsNil(t *testing.T) {
	g := NewMarotoGenerator()
	_, err := g.Render(nil)
	require.Error(t, err)
}

func TestMarotoGenerator_Render_HandlesMissingOptionalFields(t *testing.T) {
	g := NewMarotoGenerator()
	out, err := g.Render(&backend.Student{ID: 1, Name: "Stub", Email: "s@x"})
	require.NoError(t, err)
	require.NotEmpty(t, out)
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF-")))
}
