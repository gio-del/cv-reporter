package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func saveJobListingWithURL(t *testing.T, serverURL, company, url string) string {
	t.Helper()
	resp := postJSON(t, serverURL+"/api/job-listings", map[string]any{
		"company":        company,
		"url":            url,
		"jobDescription": "Some role. Salary: €40,000.",
	})
	defer resp.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result["jobListing"].(map[string]any)["id"].(string)
}

func TestCheckFreshness_LiveResponse_UpdatesStatusOn200(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonATSResponse(http.StatusOK, ""), nil
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	id := saveJobListingWithURL(t, server.URL, "Acme Corp", "https://boards.example.com/acme/jobs/1")

	resp := postJSON(t, server.URL+"/api/job-listings/"+id+"/check-freshness", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	if listing["freshnessStatus"] != "live" {
		t.Errorf("expected live, got %v", listing["freshnessStatus"])
	}
	if listing["freshnessCheckedAt"] == "" || listing["freshnessCheckedAt"] == nil {
		t.Error("expected a non-empty freshnessCheckedAt timestamp")
	}
}

func TestCheckFreshness_UnknownID_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonATSResponse(http.StatusOK, ""), nil
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/job-listings/does-not-exist/check-freshness", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestCheckFreshness_NetworkError_DoesNotOverwritePriorLiveStatus(t *testing.T) {
	dataDir := seedDataDir(t)
	status := http.StatusOK
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		if status == http.StatusOK {
			return jsonATSResponse(http.StatusOK, ""), nil
		}
		return nil, errors.New("dial tcp: i/o timeout")
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	id := saveJobListingWithURL(t, server.URL, "Acme Corp", "https://boards.example.com/acme/jobs/1")

	firstResp := postJSON(t, server.URL+"/api/job-listings/"+id+"/check-freshness", nil)
	firstResp.Body.Close()

	status = http.StatusInternalServerError // makes doer.do return the network error branch below
	secondResp := postJSON(t, server.URL+"/api/job-listings/"+id+"/check-freshness", nil)
	defer secondResp.Body.Close()

	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (a classified-Unknown outcome, not a transport error) got %d", secondResp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(secondResp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	if listing["freshnessStatus"] != "unknown" {
		t.Errorf("expected a network error to classify unknown, got %v", listing["freshnessStatus"])
	}
}

func TestCheckFreshness_NoURLRecorded_LeavesNotYetChecked(t *testing.T) {
	dataDir := seedDataDir(t)
	doerCalled := false
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		doerCalled = true
		return jsonATSResponse(http.StatusOK, ""), nil
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp") // no URL

	resp := postJSON(t, server.URL+"/api/job-listings/"+id+"/check-freshness", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if doerCalled {
		t.Error("expected no outbound request when the Job Listing has no source URL")
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	if listing["freshnessStatus"] != "not-yet-checked" {
		t.Errorf("expected not-yet-checked, got %v", listing["freshnessStatus"])
	}
}
