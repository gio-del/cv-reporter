package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func seedDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "experience", "quantyca-amplifon.md"), `---
employer: Quantyca S.p.A.
role: Data Engineer
client: Amplifon
location: Monza
start: "2024-10"
end: null
flagship: true
tags:
  - AI Platform
  - React
---

- Designed and built an AI Platform.
- Built the platform's front end in React.
`)

	writeFile(t, filepath.Join(dir, "projects", "emall.md"), `---
name: eMall
start: "2022"
end: "2022"
tags:
  - JavaScript
  - React
repo: "GitHub"
---

- Designed and implemented a system for managing charging stations.
`)

	writeFile(t, filepath.Join(dir, "profile.yaml"), `name: Test User
location: Milan, Italy
email: test@example.com
phone: "+39 333 000 0000"
linkedin: testuser
github: testuser

education:
  - degree: MSc
    institution: Test University
    program: Computer Science
    start: "2022"
    end: "2024"
    grade: "110/110"
    courses:
      - Distributed Systems

publications:
  - title: "A Paper"
    authors: "Test User"
    venue: "Test Venue"
    link: "https://example.com/paper"
    note: "A note."

awards:
  - title: "An Award"
    description: "A description."

activities:
  - title: "An Activity"
    description: "A description."

languages:
  - name: Italian
    level: Native
  - name: English
    level: Fluent
`)

	return dir
}

func TestListEntries_ReturnsEntriesSeededOnDisk(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/entries")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var entries []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	byID := map[string]map[string]any{}
	for _, e := range entries {
		byID[e["id"].(string)] = e
	}

	amplifon, ok := byID["experience/quantyca-amplifon"]
	if !ok {
		t.Fatalf("expected entry with id experience/quantyca-amplifon, got %v", byID)
	}
	if amplifon["type"] != "experience" {
		t.Errorf("expected type experience, got %v", amplifon["type"])
	}
	if amplifon["employer"] != "Quantyca S.p.A." {
		t.Errorf("expected employer Quantyca S.p.A., got %v", amplifon["employer"])
	}
	if amplifon["client"] != "Amplifon" {
		t.Errorf("expected client Amplifon, got %v", amplifon["client"])
	}
	tags, ok := amplifon["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "AI Platform" {
		t.Errorf("expected tags [AI Platform, React], got %v", amplifon["tags"])
	}

	emall, ok := byID["projects/emall"]
	if !ok {
		t.Fatalf("expected entry with id projects/emall, got %v", byID)
	}
	if emall["type"] != "project" {
		t.Errorf("expected type project, got %v", emall["type"])
	}
	if emall["name"] != "eMall" {
		t.Errorf("expected name eMall, got %v", emall["name"])
	}
}

func TestGetEntry_ReturnsFrontmatterAndBullets(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/entries/experience/quantyca-amplifon")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var entry map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		t.Fatal(err)
	}

	if entry["id"] != "experience/quantyca-amplifon" {
		t.Errorf("expected id experience/quantyca-amplifon, got %v", entry["id"])
	}
	if entry["employer"] != "Quantyca S.p.A." {
		t.Errorf("expected employer Quantyca S.p.A., got %v", entry["employer"])
	}
	if entry["role"] != "Data Engineer" {
		t.Errorf("expected role Data Engineer, got %v", entry["role"])
	}
	if entry["start"] != "2024-10" {
		t.Errorf("expected start 2024-10, got %v", entry["start"])
	}
	if entry["end"] != nil {
		t.Errorf("expected end nil, got %v", entry["end"])
	}
	bullets, ok := entry["bullets"].([]any)
	if !ok || len(bullets) != 2 {
		t.Fatalf("expected 2 bullets, got %v", entry["bullets"])
	}
	if bullets[0] != "Designed and built an AI Platform." {
		t.Errorf("expected first bullet to match source file, got %v", bullets[0])
	}
}

func TestGetEntry_UnknownID_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/entries/experience/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
