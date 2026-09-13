package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// postArchiveAction POSTs to /api/job-listings/{id}/{action} (archive or
// unarchive) and returns the decoded response body alongside its status.
func postArchiveAction(t *testing.T, serverURL, id, action string) (int, map[string]any) {
	t.Helper()
	resp := postJSON(t, serverURL+"/api/job-listings/"+id+"/"+action, nil)
	defer resp.Body.Close()
	var body map[string]any
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode, body
}

func TestArchiveJobListing_PersistsFlagAndReturnsUpdatedListing(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	status, body := postArchiveAction(t, server.URL, id, "archive")
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["jobListing"].(map[string]any)["archived"] != true {
		t.Errorf("expected the response's jobListing to be archived, got %v", body["jobListing"])
	}

	detail := getJobListingDetail(t, server.URL, id)
	if detail["jobListing"].(map[string]any)["archived"] != true {
		t.Errorf("expected the archived flag to survive a fresh read, got %v", detail["jobListing"])
	}

	content, err := os.ReadFile(filepath.Join(dataDir, "jobs", id+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "archived: true") {
		t.Errorf("expected the Job Listing file to carry archived: true, got:\n%s", content)
	}
}

func TestArchiveJobListing_Twice_IsIdempotent(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	if status, _ := postArchiveAction(t, server.URL, id, "archive"); status != http.StatusOK {
		t.Fatalf("expected first archive to return 200, got %d", status)
	}
	status, body := postArchiveAction(t, server.URL, id, "archive")
	if status != http.StatusOK {
		t.Fatalf("expected second archive to return 200, got %d", status)
	}
	if body["jobListing"].(map[string]any)["archived"] != true {
		t.Errorf("expected the listing to still be archived, got %v", body["jobListing"])
	}
}

func TestUnarchiveJobListing_ClearsFlagAndOmitsKeyOnDisk(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	before, err := os.ReadFile(filepath.Join(dataDir, "jobs", id+".md"))
	if err != nil {
		t.Fatal(err)
	}

	postArchiveAction(t, server.URL, id, "archive")
	status, body := postArchiveAction(t, server.URL, id, "unarchive")
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["jobListing"].(map[string]any)["archived"] != false {
		t.Errorf("expected the response's jobListing to be unarchived, got %v", body["jobListing"])
	}

	after, err := os.ReadFile(filepath.Join(dataDir, "jobs", id+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("expected an unarchived Job Listing file to be byte-identical to a never-archived one\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestUnarchiveJobListing_NeverArchived_IsIdempotent(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	if status, _ := postArchiveAction(t, server.URL, id, "unarchive"); status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
}

func TestArchiveAndUnarchiveJobListing_UnknownID_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	for _, action := range []string{"archive", "unarchive"} {
		if status, _ := postArchiveAction(t, server.URL, "does-not-exist", action); status != http.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", action, status)
		}
	}
	if _, err := os.Stat(filepath.Join(dataDir, "jobs", "does-not-exist.md")); !os.IsNotExist(err) {
		t.Errorf("expected no Job Listing file to be created for an unknown id, stat err: %v", err)
	}
}

func TestArchiveJobListing_LeavesApplicationUntouched(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	patchJSON(t, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": "tailoring"}).Body.Close()
	patchJSON(t, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": "sent"}).Body.Close()
	postJSON(t, server.URL+"/api/applications/"+id+"/generations", map[string]any{
		"slug":   "acme-corp-cv",
		"cvPath": "output/acme-corp-cv/cv.pdf",
	}).Body.Close()

	applicationPath := filepath.Join(dataDir, "applications", id+".md")
	before, err := os.ReadFile(applicationPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeApplication := getJobListingDetail(t, server.URL, id)["application"]

	postArchiveAction(t, server.URL, id, "archive")

	after, err := os.ReadFile(applicationPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("expected archiving to leave the Application file untouched\nbefore:\n%s\nafter:\n%s", before, after)
	}
	afterApplication := getJobListingDetail(t, server.URL, id)["application"]
	if !reflect.DeepEqual(beforeApplication, afterApplication) {
		t.Errorf("expected the Application's Status, Status history and Generation history unchanged\nbefore: %v\nafter: %v", beforeApplication, afterApplication)
	}
}

func TestGetJobListing_FileWithoutArchivedKey_IsNotArchived(t *testing.T) {
	dataDir := seedDataDir(t)
	writeFile(t, filepath.Join(dataDir, "jobs", "legacy-corp.md"), `---
company: Legacy Corp
source: manual
savedAt: "2025-01-01T00:00:00Z"
ral:
    source: n/a
---

An older Job Listing saved before archiving existed.
`)
	writeFile(t, filepath.Join(dataDir, "applications", "legacy-corp.md"), `jobListingId: legacy-corp
status: saved
method:
    kind: other
`)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	detail := getJobListingDetail(t, server.URL, "legacy-corp")
	if detail["jobListing"].(map[string]any)["archived"] != false {
		t.Errorf("expected an older Job Listing file to read as not archived, got %v", detail["jobListing"])
	}
}
