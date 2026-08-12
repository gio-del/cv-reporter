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

func postJSON(t *testing.T, url string, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestCreateEntry_ValidExperiencePayload_WritesFileAndReturns201(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	payload := map[string]any{
		"type":     "experience",
		"employer": "Acme Corp",
		"role":     "Consultant",
		"client":   "Globex",
		"location": "Remote",
		"start":    "2026-01",
		"tags":     []string{"Go"},
		"bullets":  []string{"Did the thing."},
	}

	resp := postJSON(t, server.URL+"/api/master-data/entries", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var entry map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		t.Fatal(err)
	}
	id, ok := entry["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected a generated id, got %v", entry["id"])
	}

	dir, slug, ok := splitID(id)
	if !ok || dir != "experience" {
		t.Fatalf("expected id under experience/, got %q", id)
	}

	path := filepath.Join(dataDir, "experience", slug+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
	if !bytes.Contains(content, []byte("Acme Corp")) {
		t.Errorf("expected file to contain employer, got:\n%s", content)
	}
	if !bytes.Contains(content, []byte("Did the thing.")) {
		t.Errorf("expected file to contain bullet, got:\n%s", content)
	}
}

func TestCreateEntry_ValidProjectPayload_WritesFileAndReturns201(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	payload := map[string]any{
		"type":    "project",
		"name":    "Side Project",
		"start":   "2025",
		"tags":    []string{"Rust"},
		"bullets": []string{"Built a thing."},
	}

	resp := postJSON(t, server.URL+"/api/master-data/entries", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var entry map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		t.Fatal(err)
	}
	id, ok := entry["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected a generated id, got %v", entry["id"])
	}
	dir, _, ok := splitID(id)
	if !ok || dir != "projects" {
		t.Fatalf("expected id under projects/, got %q", id)
	}
}

func TestCreateEntry_MissingRequiredField_Returns400AndNoFileCreated(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	before, err := os.ReadDir(filepath.Join(dataDir, "experience"))
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"type":  "experience",
		"start": "2026-01",
		"tags":  []string{},
	}

	resp := postJSON(t, server.URL+"/api/master-data/entries", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	after, err := os.ReadDir(filepath.Join(dataDir, "experience"))
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Errorf("expected no new file to be created, before=%d after=%d", len(before), len(after))
	}
}

func TestCreateEntry_UnknownType_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	payload := map[string]any{
		"type":  "hobby",
		"name":  "Something",
		"start": "2026",
		"tags":  []string{},
	}

	resp := postJSON(t, server.URL+"/api/master-data/entries", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// splitID mirrors the internal id shape ("<dir>/<slug>") without importing
// the internal masterdata package from the api_test (external) package.
func splitID(id string) (dir, slug string, ok bool) {
	i := bytes.IndexByte([]byte(id), '/')
	if i <= 0 || i == len(id)-1 {
		return "", "", false
	}
	return id[:i], id[i+1:], true
}
