package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// seedProjectRoot builds a temp project root containing Master Data (the
// same fixtures seedDataDir provides) plus a copy of the real
// template/*.typ files, so Render can invoke the real typst CLI against it
// without touching the repo's own output/ directory (see the PRD's Testing
// Decisions: "Render step tested by asserting a PDF file is produced and
// is one page").
func seedProjectRoot(t *testing.T) (projectRoot, dataDir string) {
	t.Helper()
	root := t.TempDir()
	dataDir = seedDataDirAt(t, filepath.Join(root, "data"))
	copyTemplate(t, root, "cv.typ")
	copyTemplate(t, root, "cover-letter.typ")
	return root, dataDir
}

func copyTemplate(t *testing.T, root, name string) {
	t.Helper()
	src := filepath.Join("..", "..", "..", "template", name)
	content, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading real template %s: %v", src, err)
	}
	dst := filepath.Join(root, "template", name)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRenderGeneration_ApprovedSelection_ProducesOnePagePDF(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	server := httptest.NewServer(api.NewRouterFull(dataDir, projectRoot, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{
		"slug": "acme-corp",
		"selection": map[string]any{
			"entries": []map[string]any{
				{
					"entryId": "experience/quantyca-amplifon",
					"reason":  "Relevant",
					"bullets": []map[string]any{
						{"sourceIndex": 0, "source": "Designed and built an AI Platform.", "rewritten": "Designed and built an AI Platform."},
					},
				},
			},
		},
	}
	resp := postJSON(t, server.URL+"/api/generations/render", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := os.ReadFile(filepath.Join(projectRoot, "output", "acme-corp", "cv.pdf"))
		t.Fatalf("expected 200, got %d (output present: %v)", resp.StatusCode, len(body) > 0)
	}

	var result generation.RenderResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.CVPageCount != 1 {
		t.Errorf("expected a one-page CV, got %d pages", result.CVPageCount)
	}

	pdfPath := filepath.Join(projectRoot, result.CVPath)
	info, err := os.Stat(pdfPath)
	if err != nil {
		t.Fatalf("expected rendered PDF to exist at %s: %v", pdfPath, err)
	}
	if info.Size() == 0 {
		t.Error("expected rendered PDF to be non-empty")
	}
}

func TestRenderGeneration_WithCoverLetter_AlsoProducesCoverLetterPDF(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	server := httptest.NewServer(api.NewRouterFull(dataDir, projectRoot, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{
		"slug":      "acme-corp",
		"selection": map[string]any{"entries": []map[string]any{}},
		"coverLetter": map[string]any{
			"body": "Dear Hiring Manager,\n\nI'm excited to apply.\n\nBest,\nCandidate",
		},
	}
	resp := postJSON(t, server.URL+"/api/generations/render", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result generation.RenderResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.CoverLetterPath == "" {
		t.Fatal("expected a coverLetterPath")
	}
	if _, err := os.Stat(filepath.Join(projectRoot, result.CoverLetterPath)); err != nil {
		t.Fatalf("expected rendered cover letter PDF to exist: %v", err)
	}

	txt, err := os.ReadFile(filepath.Join(projectRoot, "output", "acme-corp", "cover-letter.txt"))
	if err != nil {
		t.Fatalf("expected a cover-letter.txt for story 11's text download: %v", err)
	}
	if string(txt) != "Dear Hiring Manager,\n\nI'm excited to apply.\n\nBest,\nCandidate" {
		t.Errorf("expected cover-letter.txt to match the approved body, got %q", txt)
	}
}

func TestRenderGeneration_InvalidSlug_Returns400(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	server := httptest.NewServer(api.NewRouterFull(dataDir, projectRoot, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"slug": "Not Kebab Case!", "selection": map[string]any{"entries": []map[string]any{}}}
	resp := postJSON(t, server.URL+"/api/generations/render", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRenderGeneration_UnknownEntryID_Returns400(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	server := httptest.NewServer(api.NewRouterFull(dataDir, projectRoot, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{
		"slug": "acme-corp",
		"selection": map[string]any{
			"entries": []map[string]any{
				{"entryId": "experience/does-not-exist", "bullets": []map[string]any{{"sourceIndex": 0, "source": "x", "rewritten": "x"}}},
			},
		},
	}
	resp := postJSON(t, server.URL+"/api/generations/render", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
