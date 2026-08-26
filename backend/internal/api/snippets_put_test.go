package api_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func TestUpdateSnippet_ValidPayload_WritesFileAndReturns200(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	payload := map[string]any{
		"kind": "closing",
		"tags": []string{"formal"},
		"body": "I look forward to hearing from you.",
	}

	resp := putJSON(t, server.URL+"/api/master-data/cover-letter-snippets/closing-standard", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	updated, err := os.ReadFile(filepath.Join(dataDir, "cover-letter-snippets", "closing-standard.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(updated, []byte("I look forward to hearing from you.")) {
		t.Errorf("expected file to contain updated body, got:\n%s", updated)
	}
}

func TestUpdateSnippet_MissingKind_Returns400AndFileUnchanged(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	original, err := os.ReadFile(filepath.Join(dataDir, "cover-letter-snippets", "closing-standard.md"))
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"kind": "",
		"body": "Something.",
	}

	resp := putJSON(t, server.URL+"/api/master-data/cover-letter-snippets/closing-standard", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	body, _ := os.ReadFile(filepath.Join(dataDir, "cover-letter-snippets", "closing-standard.md"))
	if !bytes.Equal(body, original) {
		t.Errorf("expected file to be untouched after invalid update, got:\n%s", body)
	}
}

func TestUpdateSnippet_UnknownID_Returns404(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	payload := map[string]any{
		"kind": "closing",
		"body": "Something.",
	}

	resp := putJSON(t, server.URL+"/api/master-data/cover-letter-snippets/does-not-exist", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteSnippet_RemovesFileAndReturns204(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	path := filepath.Join(dataDir, "cover-letter-snippets", "closing-standard.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected fixture file to exist before delete: %v", err)
	}

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/master-data/cover-letter-snippets/closing-standard", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, stat err = %v", err)
	}
}

func TestDeleteSnippet_UnknownID_Returns404(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/master-data/cover-letter-snippets/does-not-exist", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
