package pdf

import (
	"fmt"
	"strconv"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/school-mgmt/go-service/internal/backend"
)

// MarotoGenerator renders a student profile into an A4 PDF using maroto/v2.
// The implementation is stateless and safe for concurrent use: every Render call constructs a fresh maroto instance.
type MarotoGenerator struct{}

// Return a ready-to-use generator.
func NewMarotoGenerator() *MarotoGenerator {
	return &MarotoGenerator{}
}

// Render lays out the student profile and returns the resulting PDF bytes.
func (g *MarotoGenerator) Render(s *backend.Student) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("nil student")
	}

	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(15).
		Build()
	m := maroto.New(cfg)

	if err := m.RegisterHeader(buildHeader(s.ID)...); err != nil {
		return nil, fmt.Errorf("register header: %w", err)
	}
	if err := m.RegisterFooter(buildFooter()...); err != nil {
		return nil, fmt.Errorf("register footer: %w", err)
	}

	m.AddRows(
		text.NewRow(14, "Student Report",
			props.Text{Size: 18, Style: fontstyle.Bold, Align: align.Center, Top: 4}),
		text.NewRow(8, fmt.Sprintf("Student ID: %d", s.ID),
			props.Text{Size: 11, Align: align.Center, Top: 2}),
	)

	m.AddRows(sectionTitle("Personal Information"))
	m.AddRows(detailRows([][2]string{
		{"Name", s.Name},
		{"Email", s.Email},
		{"Phone", deref(s.Phone)},
		{"Gender", deref(s.Gender)},
		{"Date of Birth", deref(s.DOB)},
		{"System Access", boolStr(s.SystemAccess)},
	})...)

	m.AddRows(sectionTitle("Academic Information"))
	m.AddRows(detailRows([][2]string{
		{"Class", deref(s.Class)},
		{"Section", deref(s.Section)},
		{"Roll", intPtrStr(s.Roll)},
		{"Admission Date", deref(s.AdmissionDate)},
		{"Reporter", deref(s.ReporterName)},
	})...)

	m.AddRows(sectionTitle("Family"))
	m.AddRows(detailRows([][2]string{
		{"Father Name", deref(s.FatherName)},
		{"Father Phone", deref(s.FatherPhone)},
		{"Mother Name", deref(s.MotherName)},
		{"Mother Phone", deref(s.MotherPhone)},
		{"Guardian Name", deref(s.GuardianName)},
		{"Guardian Phone", deref(s.GuardianPhone)},
		{"Guardian Relation", deref(s.RelationOfGuardian)},
	})...)

	m.AddRows(sectionTitle("Address"))
	m.AddRows(detailRows([][2]string{
		{"Current Address", deref(s.CurrentAddress)},
		{"Permanent Address", deref(s.PermanentAddress)},
	})...)

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}
	return doc.GetBytes(), nil
}

func buildHeader(studentID int) []core.Row {
	return []core.Row{
		row.New(10).Add(
			text.NewCol(6, "School Management System",
				props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Left, Top: 3}),
			text.NewCol(6, fmt.Sprintf("Student #%d", studentID),
				props.Text{Size: 9, Style: fontstyle.Italic, Align: align.Right, Top: 3}),
		),
	}
}

func buildFooter() []core.Row {
	return []core.Row{
		row.New(8).Add(
			text.NewCol(12,
				"Generated at "+time.Now().UTC().Format(time.RFC3339),
				props.Text{Size: 8, Align: align.Center, Top: 2}),
		),
	}
}

func sectionTitle(title string) core.Row {
	return text.NewRow(10, title,
		props.Text{Size: 12, Style: fontstyle.Bold, Top: 4, Align: align.Left})
}

func detailRows(items [][2]string) []core.Row {
	rows := make([]core.Row, 0, len(items))
	for _, kv := range items {
		rows = append(rows, row.New(8).Add(
			text.NewCol(4, kv[0]+":",
				props.Text{Size: 10, Style: fontstyle.Bold, Top: 2}),
			text.NewCol(8, fallback(kv[1]),
				props.Text{Size: 10, Top: 2}),
		))
	}
	return rows
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtrStr(v *int) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}

func boolStr(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func fallback(v string) string {
	if v == "" {
		return "-"
	}
	return v
}
