package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func saveListing(t *testing.T, serverURL, company, jobDescription string) string {
	t.Helper()
	resp := postJSON(t, serverURL+"/api/job-listings", map[string]any{
		"company":        company,
		"url":            "https://example.com/jobs/" + company,
		"jobDescription": jobDescription,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("seed save failed for %s: %d", company, resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	return listing["id"].(string)
}

func TestListJobListings_FilterByCompany_ReturnsOnlyMatching(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	saveListing(t, server.URL, "Acme Corp", "Go backend role.")
	saveListing(t, server.URL, "Beta Inc", "React frontend role.")

	resp, err := http.Get(server.URL + "/api/job-listings?company=acme")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 filtered result, got %d: %+v", len(got), got)
	}
}

func TestListJobListings_NoFilters_ReturnsEverything(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	saveListing(t, server.URL, "Acme Corp", "Go backend role.")
	saveListing(t, server.URL, "Beta Inc", "React frontend role.")

	resp, err := http.Get(server.URL + "/api/job-listings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var got []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 unfiltered results, got %d", len(got))
	}
}

func TestListJobListings_InvalidStatus_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/job-listings?status=not-a-real-status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", resp.StatusCode)
	}
}

func TestListJobListings_InvalidDate_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/job-listings?savedFrom=not-a-date")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid savedFrom date, got %d", resp.StatusCode)
	}
}

func TestListJobListings_FilterByStatus_ReturnsOnlyMatching(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	saveListing(t, server.URL, "Acme Corp", "Go backend role.")

	resp, err := http.Get(server.URL + "/api/job-listings?status=saved")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 result for status=saved, got %d", len(got))
	}
}
