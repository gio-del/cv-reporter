package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func TestGetGenerationFile_ServesRenderedPDF(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	outputDir := filepath.Join(projectRoot, "output", "acme-corp")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pdfBytes := []byte("%PDF-1.7 fake pdf content")
	if err := os.WriteFile(filepath.Join(outputDir, "cv.pdf"), pdfBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(api.NewRouterFull(dataDir, projectRoot, &fakeGenerationClient{}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/generations/acme-corp/cv.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("expected Content-Type application/pdf, got %q", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(pdfBytes) {
		t.Errorf("expected served bytes to match the file on disk")
	}
}

func TestGetGenerationFile_ServesCoverLetterText(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	outputDir := filepath.Join(projectRoot, "output", "acme-corp")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "cover-letter.txt"), []byte("Dear Hiring Manager,"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(api.NewRouterFull(dataDir, projectRoot, &fakeGenerationClient{}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/generations/acme-corp/cover-letter.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Dear Hiring Manager," {
		t.Errorf("expected cover letter text to be served, got %q", body)
	}
}

func TestGetGenerationFile_UnknownFilename_Returns404(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	server := httptest.NewServer(api.NewRouterFull(dataDir, projectRoot, &fakeGenerationClient{}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/generations/acme-corp/secrets.env")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for a non-allowlisted filename, got %d", resp.StatusCode)
	}
}

func TestGetGenerationFile_PathTraversalAttempt_Returns404(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	server := httptest.NewServer(api.NewRouterFull(dataDir, projectRoot, &fakeGenerationClient{}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/generations/..%2f..%2fetc/cv.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected a path traversal attempt not to succeed, got %d", resp.StatusCode)
	}
}

func TestGetGenerationFile_MissingFile_Returns404(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	server := httptest.NewServer(api.NewRouterFull(dataDir, projectRoot, &fakeGenerationClient{}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/generations/never-rendered/cv.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
