package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func deleteRequest(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestDeleteJobListing_RemovesFilesAndReturns204(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp := deleteRequest(t, server.URL+"/api/job-listings/"+id)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	if _, err := os.Stat(filepath.Join(dataDir, "jobs", id+".md")); !os.IsNotExist(err) {
		t.Fatalf("expected job listing file to be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "applications", id+".md")); !os.IsNotExist(err) {
		t.Fatalf("expected application file to be removed, stat err = %v", err)
	}

	getResp, err := http.Get(server.URL + "/api/job-listings/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected a follow-up GET to 404, got %d", getResp.StatusCode)
	}
}

func TestDeleteJobListing_UnknownID_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	resp := deleteRequest(t, server.URL+"/api/job-listings/does-not-exist")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteJobListing_RemovesLogoFile(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(bytes.NewReader(fixturePNG)),
		}, nil
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	saveResp := postJSON(t, server.URL+"/api/job-listings/from-extension", map[string]any{
		"company":     "Acme Corp",
		"description": "Go backend engineer.",
		"logoUrl":     "https://media.linkedin.com/logo.png",
	})
	var saved struct {
		JobListing struct {
			ID   string `json:"id"`
			Logo string `json:"logo"`
		} `json:"jobListing"`
	}
	if err := json.NewDecoder(saveResp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	saveResp.Body.Close()

	logoPath := filepath.Join(dataDir, "jobs", saved.JobListing.Logo)
	if _, err := os.Stat(logoPath); err != nil {
		t.Fatalf("expected fixture logo file to exist before delete: %v", err)
	}

	resp := deleteRequest(t, server.URL+"/api/job-listings/"+saved.JobListing.ID)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if _, err := os.Stat(logoPath); !os.IsNotExist(err) {
		t.Fatalf("expected logo file to be removed, stat err = %v", err)
	}
}

func TestDeleteJobListing_WithGenerationHistory_StillSucceeds(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	genResp := postJSON(t, server.URL+"/api/applications/"+id+"/generations", map[string]any{
		"slug":   "acme-corp-2026-01-01",
		"cvPath": "output/acme-corp-2026-01-01/cv.pdf",
	})
	defer genResp.Body.Close()
	if genResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 recording the generation, got %d", genResp.StatusCode)
	}

	resp := deleteRequest(t, server.URL+"/api/job-listings/"+id)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}
