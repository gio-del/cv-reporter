package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

const snippetOutOfBand = `---
kind: opening
tags:
  - Go
---

A paragraph the skill rewrote while the form was open.
`

func seedSnippet(t *testing.T, dataDir string) string {
	t.Helper()
	path := filepath.Join(dataDir, "cover-letter-snippets", "opening.md")
	writeFile(t, path, `---
kind: opening
tags:
  - React
---

The paragraph the user opened.
`)
	return path
}

func TestGetSnippet_ReturnsNonEmptyVersionToken(t *testing.T) {
	dataDir := seedDataDir(t)
	seedSnippet(t, dataDir)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	getVersion(t, server.URL+"/api/master-data/cover-letter-snippets/opening")
}

func TestUpdateSnippet_WithStaleVersion_Returns409AndLeavesFileUntouched(t *testing.T) {
	dataDir := seedDataDir(t)
	path := seedSnippet(t, dataDir)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/cover-letter-snippets/opening"
	version := getVersion(t, url)
	writeOutOfBand(t, path, snippetOutOfBand)

	payload := map[string]any{"kind": "opening", "tags": []string{"React"}, "body": "The user's edit."}
	resp := doJSON(t, http.MethodPut, url, payload, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, path); got != snippetOutOfBand {
		t.Errorf("expected the file to be byte-identical to its pre-request state, got:\n%s", got)
	}
}

func TestUpdateSnippet_WithCurrentVersion_SucceedsAndReturnsAFreshToken(t *testing.T) {
	dataDir := seedDataDir(t)
	seedSnippet(t, dataDir)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/cover-letter-snippets/opening"
	version := getVersion(t, url)

	payload := map[string]any{"kind": "opening", "tags": []string{"React"}, "body": "The user's edit."}
	resp := doJSON(t, http.MethodPut, url, payload, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if next := decodeVersion(t, resp); next == "" || next == version {
		t.Errorf("expected a fresh version token in the write response, got %q", next)
	}
}

func TestUpdateSnippet_WithNoVersion_IsUnconditional(t *testing.T) {
	dataDir := seedDataDir(t)
	path := seedSnippet(t, dataDir)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	writeOutOfBand(t, path, snippetOutOfBand)

	payload := map[string]any{"kind": "opening", "tags": []string{"React"}, "body": "The user's edit."}
	resp := doJSON(t, http.MethodPut, server.URL+"/api/master-data/cover-letter-snippets/opening", payload, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDeleteSnippet_WithStaleVersion_Returns409AndTheFileSurvives(t *testing.T) {
	dataDir := seedDataDir(t)
	path := seedSnippet(t, dataDir)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/cover-letter-snippets/opening"
	version := getVersion(t, url)
	writeOutOfBand(t, path, snippetOutOfBand)

	resp := doJSON(t, http.MethodDelete, url, nil, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, path); got != snippetOutOfBand {
		t.Errorf("expected the file to survive the refused delete unchanged, got:\n%s", got)
	}
}

func TestUpdateSnippet_RecordDeletedOutOfBand_Returns404NotConflict(t *testing.T) {
	dataDir := seedDataDir(t)
	path := seedSnippet(t, dataDir)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/cover-letter-snippets/opening"
	version := getVersion(t, url)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{"kind": "opening", "tags": []string{"React"}, "body": "The user's edit."}
	resp := doJSON(t, http.MethodPut, url, payload, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
