package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func seedSnippetsDataDir(t *testing.T) string {
	t.Helper()
	dir := seedDataDir(t)

	writeFile(t, filepath.Join(dir, "cover-letter-snippets", "opening-ai-platforms.md"), `---
kind: opening
tags:
  - AI Platform
---

I'm excited to apply for this role because it sits squarely at the
intersection of AI platform engineering and data infrastructure.
`)

	writeFile(t, filepath.Join(dir, "cover-letter-snippets", "closing-standard.md"), `---
kind: closing
---

Thank you for your time and consideration; I'd welcome the chance to
discuss how my background fits this role.
`)

	return dir
}

func TestListSnippets_ReturnsSnippetsSeededOnDisk(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/cover-letter-snippets")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var snippets []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
		t.Fatal(err)
	}

	if len(snippets) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(snippets))
	}

	byID := map[string]map[string]any{}
	for _, s := range snippets {
		byID[s["id"].(string)] = s
	}

	opening, ok := byID["opening-ai-platforms"]
	if !ok {
		t.Fatalf("expected snippet with id opening-ai-platforms, got %v", byID)
	}
	if opening["kind"] != "opening" {
		t.Errorf("expected kind opening, got %v", opening["kind"])
	}
	tags, ok := opening["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != "AI Platform" {
		t.Errorf("expected tags [AI Platform], got %v", opening["tags"])
	}
}

func TestListSnippets_IncludesLastUsedAtFromRecordedGenerations(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	resp := postJSON(t, server.URL+"/api/applications/"+id+"/generations", map[string]any{
		"slug":             "acme-corp",
		"cvPath":           "output/acme-corp/cv.pdf",
		"coverLetterPath":  "output/acme-corp/cover-letter.pdf",
		"sourceSnippetIds": []string{"opening-ai-platforms"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 recording the Generation, got %d", resp.StatusCode)
	}

	listResp, err := http.Get(server.URL + "/api/master-data/cover-letter-snippets")
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()

	var snippets []map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&snippets); err != nil {
		t.Fatal(err)
	}

	byID := map[string]map[string]any{}
	for _, s := range snippets {
		byID[s["id"].(string)] = s
	}

	used := byID["opening-ai-platforms"]
	if used["lastUsedAt"] == nil || used["lastUsedAt"] == "" {
		t.Errorf("expected opening-ai-platforms to have a non-empty lastUsedAt, got %v", used["lastUsedAt"])
	}

	unused := byID["closing-standard"]
	if v, ok := unused["lastUsedAt"]; ok && v != nil {
		t.Errorf("expected closing-standard (never used) to have no lastUsedAt, got %v", v)
	}
}

func TestGetSnippet_ReturnsFrontmatterAndBody(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/cover-letter-snippets/closing-standard")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var snippet map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&snippet); err != nil {
		t.Fatal(err)
	}

	if snippet["id"] != "closing-standard" {
		t.Errorf("expected id closing-standard, got %v", snippet["id"])
	}
	if snippet["kind"] != "closing" {
		t.Errorf("expected kind closing, got %v", snippet["kind"])
	}
	body, ok := snippet["body"].(string)
	if !ok || body == "" {
		t.Fatalf("expected non-empty body, got %v", snippet["body"])
	}
	if want := "Thank you for your time"; !strings.Contains(body, want) {
		t.Errorf("expected body to contain %q, got %q", want, body)
	}
}

func TestGetSnippet_UnknownID_Returns404(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/cover-letter-snippets/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
