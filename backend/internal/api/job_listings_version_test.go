package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// deleteJobListing issues DELETE /api/job-listings/{id} carrying the Job
// Listing's token as If-Match and the Application's token as
// Application-If-Match, each only when non-empty — the FE's exact shape.
func deleteJobListing(t *testing.T, serverURL, id, jobListingVersion, applicationVersion string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, serverURL+"/api/job-listings/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if jobListingVersion != "" {
		req.Header.Set("If-Match", jobListingVersion)
	}
	if applicationVersion != "" {
		req.Header.Set("Application-If-Match", applicationVersion)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestDeleteJobListing_WithCurrentVersions_Succeeds(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	jobListing, application := listingVersions(t, server.URL, id)

	resp := deleteJobListing(t, server.URL, id, jobListing, application)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if fileExists(filepath.Join(dataDir, "jobs", id+".md")) {
		t.Error("expected the Job Listing file to be removed")
	}
}

func TestDeleteJobListing_WithStaleJobListingVersion_Returns409AndBothFilesSurvive(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	jobListing, application := listingVersions(t, server.URL, id)

	listingPath := filepath.Join(dataDir, "jobs", id+".md")
	changed := readFile(t, listingPath) + "\nEdited by hand.\n"
	writeOutOfBand(t, listingPath, changed)

	resp := deleteJobListing(t, server.URL, id, jobListing, application)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, listingPath); got != changed {
		t.Errorf("expected the Job Listing file to survive unchanged, got:\n%s", got)
	}
	if !fileExists(filepath.Join(dataDir, "applications", id+".md")) {
		t.Error("expected the Application file to survive a refused delete")
	}
}

// Deleting a Job Listing destroys its Application too, so a Status that
// moved on since the list was loaded must also refuse the delete (story
// 19).
func TestDeleteJobListing_WithStaleApplicationVersion_Returns409AndBothFilesSurvive(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	jobListing, application := listingVersions(t, server.URL, id)

	applicationPath := filepath.Join(dataDir, "applications", id+".md")
	writeOutOfBand(t, applicationPath, applicationOutOfBand)

	resp := deleteJobListing(t, server.URL, id, jobListing, application)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, applicationPath); got != applicationOutOfBand {
		t.Errorf("expected the Application file to survive unchanged, got:\n%s", got)
	}
	if !fileExists(filepath.Join(dataDir, "jobs", id+".md")) {
		t.Error("expected the Job Listing file to survive a refused delete")
	}
}

func TestDeleteJobListing_WithNoVersion_IsUnconditional(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	writeOutOfBand(t, filepath.Join(dataDir, "applications", id+".md"), applicationOutOfBand)

	resp := deleteJobListing(t, server.URL, id, "", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestDeleteJobListing_DeletedOutOfBand_Returns404NotConflict(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	jobListing, application := listingVersions(t, server.URL, id)
	removeFile(t, filepath.Join(dataDir, "jobs", id+".md"))

	resp := deleteJobListing(t, server.URL, id, jobListing, application)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
