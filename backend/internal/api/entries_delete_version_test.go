package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func TestDeleteEntry_WithCurrentVersion_Succeeds(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	url := server.URL + "/api/master-data/entries/" + entryID
	version := getVersion(t, url)

	resp := doJSON(t, http.MethodDelete, url, nil, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "experience", "quantyca-amplifon.md")); !os.IsNotExist(err) {
		t.Fatalf("expected the file to be removed, stat err = %v", err)
	}
}

func TestDeleteEntry_WithStaleVersion_Returns409AndTheFileSurvives(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	url := server.URL + "/api/master-data/entries/" + entryID
	path := filepath.Join(dataDir, "experience", "quantyca-amplifon.md")
	version := getVersion(t, url)

	outOfBand := `---
employer: Quantyca S.p.A.
start: "2024-10"
---

- A bullet the skill added after the list was loaded.
`
	writeOutOfBand(t, path, outOfBand)

	resp := doJSON(t, http.MethodDelete, url, nil, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, path); got != outOfBand {
		t.Errorf("expected the file to survive the refused delete unchanged, got:\n%s", got)
	}
}

func TestDeleteEntry_WithNoVersion_IsUnconditional(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	path := filepath.Join(dataDir, "experience", "quantyca-amplifon.md")
	writeOutOfBand(t, path, `---
employer: Quantyca S.p.A.
start: "2024-10"
---

- Changed underneath.
`)

	resp := doJSON(t, http.MethodDelete, server.URL+"/api/master-data/entries/"+entryID, nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}
