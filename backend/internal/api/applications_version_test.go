package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// listingVersions reads GET /api/job-listings and returns the version
// tokens of the Job Listing and of its Application. An Application and its
// Job Listing are two separate files sharing one id, so they carry two
// separate tokens.
func listingVersions(t *testing.T, serverURL, id string) (jobListing, application string) {
	t.Helper()
	resp, err := http.Get(serverURL + "/api/job-listings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var listings []map[string]map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		t.Fatal(err)
	}
	for _, l := range listings {
		if l["jobListing"]["id"] != id {
			continue
		}
		jobListing, _ = l["jobListing"]["version"].(string)
		application, _ = l["application"]["version"].(string)
		return jobListing, application
	}
	t.Fatalf("job listing %q not found in the list response", id)
	return "", ""
}

const applicationOutOfBand = `jobListingId: acme-corp
status: interviewing
method:
    kind: portal
`

func TestListJobListings_ReturnsATokenForBothTheListingAndItsApplication(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	jobListing, application := listingVersions(t, server.URL, id)
	if jobListing == "" {
		t.Error("expected a version token on the Job Listing")
	}
	if application == "" {
		t.Error("expected a version token on the Application")
	}
	if jobListing == application {
		t.Error("expected the two files to carry two distinct tokens")
	}
}

func TestUpdateApplicationStatus_WithCurrentVersion_Succeeds(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	_, version := listingVersions(t, server.URL, id)

	resp := doJSON(t, http.MethodPatch, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": "tailoring"}, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if next := decodeVersion(t, resp); next == "" || next == version {
		t.Errorf("expected a fresh version token in the write response, got %q", next)
	}
}

func TestUpdateApplicationStatus_WithStaleVersion_Returns409AndLeavesFileUntouched(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	_, version := listingVersions(t, server.URL, id)

	// A second tab (or the skill) moved the Application on; the stale tab
	// must not be able to drag it backwards through the funnel.
	path := filepath.Join(dataDir, "applications", id+".md")
	writeOutOfBand(t, path, applicationOutOfBand)

	resp := doJSON(t, http.MethodPatch, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": "tailoring"}, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, path); got != applicationOutOfBand {
		t.Errorf("expected the file to be byte-identical to its pre-request state, got:\n%s", got)
	}
}

func TestUpdateApplicationStatus_WithNoVersion_IsUnconditional(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp := doJSON(t, http.MethodPatch, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": "tailoring"}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdateApplicationStatus_ApplicationDeletedOutOfBand_Returns404NotConflict(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	_, version := listingVersions(t, server.URL, id)

	removeFile(t, filepath.Join(dataDir, "applications", id+".md"))

	resp := doJSON(t, http.MethodPatch, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": "tailoring"}, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateApplicationMethod_WithStaleVersion_Returns409AndLeavesFileUntouched(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	_, version := listingVersions(t, server.URL, id)

	path := filepath.Join(dataDir, "applications", id+".md")
	writeOutOfBand(t, path, applicationOutOfBand)

	payload := map[string]any{"kind": "email", "value": "jobs@acme.example"}
	resp := doJSON(t, http.MethodPatch, server.URL+"/api/applications/"+id+"/method", payload, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, path); got != applicationOutOfBand {
		t.Errorf("expected the file to be byte-identical to its pre-request state, got:\n%s", got)
	}
}

func TestUpdateApplicationContact_WithStaleVersion_Returns409AndLeavesFileUntouched(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	_, version := listingVersions(t, server.URL, id)

	path := filepath.Join(dataDir, "applications", id+".md")
	writeOutOfBand(t, path, applicationOutOfBand)

	payload := map[string]any{"name": "A Recruiter", "email": "recruiter@acme.example"}
	resp := doJSON(t, http.MethodPatch, server.URL+"/api/applications/"+id+"/contact", payload, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, path); got != applicationOutOfBand {
		t.Errorf("expected the file to be byte-identical to its pre-request state, got:\n%s", got)
	}
}

// Recording a Generation is deliberately out of the check: it appends to
// the Generation history rather than rewriting a field the user has been
// holding open, and a bookkeeping conflict must never block the end of a
// Tailoring run (issue #89's Out of Scope).
func TestRecordGeneration_StaysUnconditionalAfterAnOutOfBandChange(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	writeOutOfBand(t, filepath.Join(dataDir, "applications", id+".md"), applicationOutOfBand)

	payload := map[string]any{"slug": "acme-corp-2026-09", "cvPath": "output/acme-corp-2026-09/cv.pdf"}
	resp := doJSON(t, http.MethodPost, server.URL+"/api/applications/"+id+"/generations", payload, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestGetJobListing_ReturnsTheJobListingToken(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	fromList, _ := listingVersions(t, server.URL, id)

	if got := getVersion(t, server.URL+"/api/job-listings/"+id); got != fromList {
		t.Errorf("expected the detail read to carry the same Job Listing token as the list, got %q want %q", got, fromList)
	}
}
