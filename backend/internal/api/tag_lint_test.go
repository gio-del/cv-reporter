package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// seedTagLintDataDir writes its own Entries (rather than reusing
// seedDataDir) so the tag-lint fixture data is self-contained: a
// case-only/alias duplicate ("Go"/"go"), an edit-distance typo
// ("Kubernets"/"Kubernetes"), and a singleton ("Python").
func seedTagLintDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "experience", "acme-backend.md"), `---
employer: Acme Corp
role: Backend Engineer
start: "2024-01"
end: null
tags:
  - Go
  - Kubernetes
  - Python
---

- Built backend services.
`)

	writeFile(t, filepath.Join(dir, "projects", "side-project.md"), `---
name: Side Project
start: "2023"
end: "2023"
tags:
  - go
  - Kubernets
---

- Built a side project.
`)

	return dir
}

func TestTagLint_GroupsNearDuplicatesAcrossEntries(t *testing.T) {
	dataDir := seedTagLintDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/tag-lint")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var report struct {
		Groups []struct {
			Key         string `json:"key"`
			Confidence  string `json:"confidence"`
			Occurrences []struct {
				Tag       string `json:"tag"`
				EntryID   string `json:"entryId"`
				EntryType string `json:"entryType"`
			} `json:"occurrences"`
		} `json:"groups"`
		Singletons []struct {
			Tag       string `json:"tag"`
			EntryID   string `json:"entryId"`
			EntryType string `json:"entryType"`
		} `json:"singletons"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}

	if len(report.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d: %+v", len(report.Groups), report.Groups)
	}

	byKey := map[string]struct {
		Confidence string
		Count      int
	}{}
	for _, g := range report.Groups {
		byKey[g.Key] = struct {
			Confidence string
			Count      int
		}{g.Confidence, len(g.Occurrences)}
	}

	goGroup, ok := byKey["go"]
	if !ok || goGroup.Confidence != "confident" || goGroup.Count != 2 {
		t.Errorf("expected confident 'go' group with 2 occurrences, got %+v (ok=%v)", goGroup, ok)
	}

	k8sGroup, ok := byKey["kubernetes"]
	if !ok || k8sGroup.Confidence != "suggested" || k8sGroup.Count != 2 {
		t.Errorf("expected suggested 'kubernetes' group with 2 occurrences, got %+v (ok=%v)", k8sGroup, ok)
	}

	if len(report.Singletons) != 1 || report.Singletons[0].Tag != "Python" {
		t.Fatalf("expected Python as the only singleton, got %+v", report.Singletons)
	}
	if report.Singletons[0].EntryID != "experience/acme-backend" {
		t.Errorf("expected Python's entryId to be experience/acme-backend, got %v", report.Singletons[0].EntryID)
	}
	if report.Singletons[0].EntryType != "experience" {
		t.Errorf("expected Python's entryType to be experience, got %v", report.Singletons[0].EntryType)
	}

	// Spot-check that entry type is populated for a group occurrence too.
	for _, g := range report.Groups {
		if g.Key != "go" {
			continue
		}
		for _, o := range g.Occurrences {
			if o.Tag == "go" && o.EntryType != "project" {
				t.Errorf("expected lowercase 'go' occurrence's entryType to be project, got %v", o.EntryType)
			}
			if o.Tag == "Go" && o.EntryType != "experience" {
				t.Errorf("expected 'Go' occurrence's entryType to be experience, got %v", o.EntryType)
			}
		}
	}
}

func TestTagLint_NoEntries_ReturnsEmptyReport(t *testing.T) {
	dataDir := t.TempDir()
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/tag-lint")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var report struct {
		Groups     []any `json:"groups"`
		Singletons []any `json:"singletons"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if len(report.Groups) != 0 || len(report.Singletons) != 0 {
		t.Fatalf("expected an empty report, got %+v", report)
	}
}
