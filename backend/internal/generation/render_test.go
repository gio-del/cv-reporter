package generation

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
)

// requireBinary skips t unless name is on PATH — for the smoke test below,
// which needs the real typst and pdftotext binaries rather than fakes
// (PRD's Testing Decisions: "skipped if either binary isn't on PATH,
// matching how other tests here would need to handle optional local
// tooling").
func requireBinary(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not on PATH, skipping", name)
	}
}

// TestParsabilityCheck_RealTypstAndPdftotextPipeline_ReportsOK is the PRD's
// thin integration smoke test: it renders a fixture cvData through the
// real typst compile -> pdftotext pipeline once, confirming the two tools
// are wired together correctly end to end. It is deliberately not where
// the comparison logic itself is exercised — see TestCheckParsability in
// parsability_test.go for that, against fixture strings instead of a real
// PDF.
func TestParsabilityCheck_RealTypstAndPdftotextPipeline_ReportsOK(t *testing.T) {
	requireBinary(t, "typst")
	requireBinary(t, "pdftotext")

	projectRoot := t.TempDir()
	copyTemplateFixture(t, projectRoot, "cv.typ")
	if err := os.MkdirAll(filepath.Join(projectRoot, "output", "smoke-test"), 0o755); err != nil {
		t.Fatal(err)
	}

	cv := cvData{
		Name:      "Jane Doe",
		Location:  "Milan, Italy",
		Email:     "jane@example.com",
		Phone:     "+39 000 000 000",
		LinkedIn:  "janedoe",
		GitHub:    "janedoe",
		Education: []masterdata.Education{},
		Experience: []cvExperience{
			{
				Employer: "Acme Corp",
				Role:     "Senior Engineer",
				Start:    "2022",
				End:      nil,
				Bullets:  []string{"Shipped things."},
			},
		},
		Projects:     []cvProject{},
		TechStack:    []string{},
		Publications: []masterdata.Publication{},
		Awards:       []masterdata.Award{},
		Activities:   []masterdata.Activity{},
		Languages:    []masterdata.Language{},
	}

	pdfRelPath, err := renderTypst(projectRoot, "template/cv.typ", "smoke-test", "data.json", "cv.pdf", cv)
	if err != nil {
		t.Fatalf("renderTypst: %v", err)
	}

	result := checkPDFParsability(filepath.Join(projectRoot, pdfRelPath), cvExpectedFields(cv))
	if result.Status != ParsabilityOK {
		t.Errorf("expected ParsabilityOK, got %+v", result)
	}
}

// copyTemplateFixture copies the repo's real template/<name> into
// projectRoot/template/<name>, mirroring internal/api's copyTemplate
// helper — Render's smoke test needs the real template, not a
// reimplementation, so a change to cv.typ's structure is caught here too.
func copyTemplateFixture(t *testing.T, projectRoot, name string) {
	t.Helper()
	src := filepath.Join("..", "..", "..", "template", name)
	content, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading real template %s: %v", src, err)
	}
	dst := filepath.Join(projectRoot, "template", name)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, content, 0o644); err != nil {
		t.Fatal(err)
	}
}
