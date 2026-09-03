package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func TestCaptureJobListingFromExtension_ValidPayload_WritesFilesAndCreatesSavedApplication(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{
		"title":       "Backend Engineer",
		"company":     "Acme Corp",
		"location":    "Milan, Italy",
		"url":         "https://www.linkedin.com/jobs/view/12345",
		"description": "Go backend engineer, remote friendly.",
	}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	listing, ok := result["jobListing"].(map[string]any)
	if !ok {
		t.Fatalf("expected a jobListing object, got %v", result["jobListing"])
	}
	id, ok := listing["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected a generated jobListing id, got %v", listing["id"])
	}
	if listing["company"] != "Acme Corp" {
		t.Errorf("expected company Acme Corp, got %v", listing["company"])
	}
	if listing["url"] != "https://www.linkedin.com/jobs/view/12345" {
		t.Errorf("expected url to carry through, got %v", listing["url"])
	}

	application, ok := result["application"].(map[string]any)
	if !ok {
		t.Fatalf("expected an application object, got %v", result["application"])
	}
	if application["jobListingId"] != id {
		t.Errorf("expected application jobListingId %q, got %v", id, application["jobListingId"])
	}
	if application["status"] != "saved" {
		t.Errorf("expected application status saved, got %v", application["status"])
	}

	jobFile, err := os.ReadFile(filepath.Join(dataDir, "jobs", id+".md"))
	if err != nil {
		t.Fatalf("expected job listing file to exist: %v", err)
	}
	if !strings.Contains(string(jobFile), "Go backend engineer, remote friendly.") {
		t.Errorf("expected job listing file to contain the captured description, got:\n%s", jobFile)
	}
}

func TestCaptureJobListingFromExtension_MissingCompany_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"description": "Go backend engineer, remote friendly."}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCaptureJobListingFromExtension_MissingDescription_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp"}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// The caller is a browser extension's background script running on a
// moz-extension:// or chrome-extension:// origin — the browser CORS-checks
// that cross-origin fetch regardless of the extension's declared
// host_permissions on at least some Firefox versions (observed in manual
// verification), so this endpoint answers with permissive CORS headers
// rather than depending on that exemption holding.
func TestCaptureJobListingFromExtension_PostResponseIncludesCORSHeader(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp", "description": "Go backend engineer."}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
}

func TestCaptureJobListingFromExtension_OptionsPreflight_Returns204WithCORSHeaders(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodOptions, server.URL+"/api/job-listings/from-extension", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Errorf("expected Access-Control-Allow-Methods to include POST, got %q", got)
	}
}
