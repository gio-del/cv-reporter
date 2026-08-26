package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
	"github.com/gio-del/cv-reporter/backend/internal/generation"
	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

func TestCreateJobListing_StatedRAL_WritesFilesAndCreatesSavedApplication(t *testing.T) {
	dataDir := seedDataDir(t)
	estimateCalled := false
	client := &fakeGenerationClient{
		estimateRAL: func(ctx context.Context, jobDescription string) (generation.RALRange, error) {
			estimateCalled = true
			return generation.RALRange{Source: generation.RALSourceNA}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	payload := map[string]any{
		"company":        "Acme Corp",
		"url":            "https://acme.example/jobs/42",
		"jobDescription": "Go backend engineer. Salary: €50,000 - €60,000.",
	}
	resp := postJSON(t, server.URL+"/api/job-listings", payload)
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
	if listing["source"] != "manual" {
		t.Errorf("expected source manual, got %v", listing["source"])
	}
	ral, ok := listing["ral"].(map[string]any)
	if !ok || ral["source"] != "stated" {
		t.Fatalf("expected stated RAL parsed from the job description, got %v", listing["ral"])
	}
	if estimateCalled {
		t.Error("expected EstimateRAL not to be called when a RAL figure is stated in the job description")
	}

	application, ok := result["application"].(map[string]any)
	if !ok {
		t.Fatalf("expected an application object, got %v", result["application"])
	}
	if application["id"] != id {
		t.Errorf("expected application id to match jobListing id %q, got %v", id, application["id"])
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
	if !bytes.Contains(jobFile, []byte("Acme Corp")) || !bytes.Contains(jobFile, []byte("Go backend engineer")) {
		t.Errorf("expected job listing file to contain company and job description, got:\n%s", jobFile)
	}

	applicationFile, err := os.ReadFile(filepath.Join(dataDir, "applications", id+".md"))
	if err != nil {
		t.Fatalf("expected application file to exist: %v", err)
	}
	if !bytes.Contains(applicationFile, []byte("saved")) {
		t.Errorf("expected application file to record status saved, got:\n%s", applicationFile)
	}
}

func TestCreateJobListing_InfersApplicationMethodFromJobDescription(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{
		inferApplicationMethod: func(ctx context.Context, jobDescription string) (tracking.ApplicationMethod, error) {
			if jobDescription != "Email your CV to jobs@acme.example." {
				t.Errorf("expected job description to reach the client, got %q", jobDescription)
			}
			return tracking.ApplicationMethod{Kind: tracking.MethodEmail, Value: "jobs@acme.example"}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp", "jobDescription": "Email your CV to jobs@acme.example."}
	resp := postJSON(t, server.URL+"/api/job-listings", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	application := result["application"].(map[string]any)
	method, ok := application["method"].(map[string]any)
	if !ok {
		t.Fatalf("expected an application method object, got %v", application["method"])
	}
	if method["kind"] != "email" || method["value"] != "jobs@acme.example" {
		t.Errorf("expected inferred email method, got %v", method)
	}
}

func TestCreateJobListing_NoStatedRAL_CallsClientEstimateRAL(t *testing.T) {
	dataDir := seedDataDir(t)
	estimateCalled := false
	client := &fakeGenerationClient{
		estimateRAL: func(ctx context.Context, jobDescription string) (generation.RALRange, error) {
			estimateCalled = true
			min, max := 40000, 50000
			return generation.RALRange{Min: &min, Max: &max, Currency: "EUR", Source: generation.RALSourceEstimated}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp", "jobDescription": "Go backend engineer, remote."}
	resp := postJSON(t, server.URL+"/api/job-listings", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if !estimateCalled {
		t.Error("expected EstimateRAL to be called when no RAL figure is stated in the job description")
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	ral := listing["ral"].(map[string]any)
	if ral["source"] != "estimated" {
		t.Errorf("expected estimated RAL, got %v", ral["source"])
	}
}

func TestCreateJobListing_JobDescriptionURL_FetchesText(t *testing.T) {
	dataDir := seedDataDir(t)
	jdServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body><p>Go backend engineer.</p></body></html>"))
	}))
	defer jdServer.Close()

	client := &fakeGenerationClient{}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp", "jobDescriptionUrl": jdServer.URL}
	resp := postJSON(t, server.URL+"/api/job-listings", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	if listing["jobDescription"] != "Go backend engineer." {
		t.Errorf("expected fetched+stripped job description, got %v", listing["jobDescription"])
	}
}

func TestCreateJobListing_MissingCompany_Returns400AndNoFilesCreated(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"jobDescription": "Go backend engineer."}
	resp := postJSON(t, server.URL+"/api/job-listings", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	assertNoFilesIn(t, filepath.Join(dataDir, "jobs"))
}

func TestCreateJobListing_MissingJobDescription_Returns400AndNoFilesCreated(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp"}
	resp := postJSON(t, server.URL+"/api/job-listings", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	assertNoFilesIn(t, filepath.Join(dataDir, "jobs"))
}

func TestListJobListings_ReturnsEachWithItsApplicationNewestFirst(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "First Corp",
		"jobDescription": "Role one. Salary: €40,000.",
	}).Body.Close()
	postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Second Corp",
		"jobDescription": "Role two. Salary: €50,000.",
	}).Body.Close()

	resp, err := http.Get(server.URL + "/api/job-listings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 job listings, got %d", len(results))
	}

	first := results[0]["jobListing"].(map[string]any)
	if first["company"] != "Second Corp" {
		t.Errorf("expected the most recently saved listing (Second Corp) first, got %v", first["company"])
	}

	application := results[0]["application"].(map[string]any)
	if application["status"] != "saved" {
		t.Errorf("expected application status saved, got %v", application["status"])
	}
	if application["jobListingId"] != first["id"] {
		t.Errorf("expected application jobListingId to match jobListing id, got %v vs %v", application["jobListingId"], first["id"])
	}

	ral := first["ral"].(map[string]any)
	if ral["source"] != "stated" {
		t.Errorf("expected stated RAL on the list view, got %v", ral["source"])
	}
}

func TestListJobListings_NoneSaved_ReturnsEmptyArray(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/job-listings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected no job listings, got %v", results)
	}
}

func TestGetJobListing_ReturnsSavedRecord(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp, err := http.Get(server.URL + "/api/job-listings/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var listing map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		t.Fatal(err)
	}
	if listing["company"] != "Acme Corp" {
		t.Errorf("expected company Acme Corp, got %v", listing["company"])
	}
}

func TestGetJobListing_UnknownID_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/job-listings/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func assertNoFilesIn(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files in %s, found %v", dir, entries)
	}
}
