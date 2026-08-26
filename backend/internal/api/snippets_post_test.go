package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func TestCreateSnippet_ValidPayload_WritesFileAndReturns201(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	payload := map[string]any{
		"kind": "why-this-company",
		"tags": []string{"AI"},
		"body": "I've followed this company's AI platform work for years.",
	}

	resp := postJSON(t, server.URL+"/api/master-data/cover-letter-snippets", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var snippet map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&snippet); err != nil {
		t.Fatal(err)
	}
	id, ok := snippet["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected a generated id, got %v", snippet["id"])
	}

	path := filepath.Join(dataDir, "cover-letter-snippets", id+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
	if !bytes.Contains(content, []byte("why-this-company")) {
		t.Errorf("expected file to contain kind, got:\n%s", content)
	}
	if !bytes.Contains(content, []byte("I've followed this company's AI platform work for years.")) {
		t.Errorf("expected file to contain body, got:\n%s", content)
	}
}

func TestCreateSnippet_MissingKind_Returns400AndNoFileCreated(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	before, err := os.ReadDir(filepath.Join(dataDir, "cover-letter-snippets"))
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"body": "Some prose.",
	}

	resp := postJSON(t, server.URL+"/api/master-data/cover-letter-snippets", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	after, err := os.ReadDir(filepath.Join(dataDir, "cover-letter-snippets"))
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Errorf("expected no new file to be created, before=%d after=%d", len(before), len(after))
	}
}
