package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// TestListJobListings_SortByRALDesc_OrdersNumericFirstThenTrailsNonNumeric
// confirms sort=ral&order=desc threads through GET /api/job-listings
// end-to-end (issue #51's handler-level integration test), against a small
// fixture set covering a stated, an estimated (via a JD with no stated
// figure), and an unresolved-RAL listing.
func TestListJobListings_SortByRALDesc_OrdersNumericFirstThenTrailsNonNumeric(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{}
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: client}))
	defer server.Close()

	postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Low Corp",
		"jobDescription": "Role. Salary: €40,000.",
	}).Body.Close()
	postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "High Corp",
		"jobDescription": "Role. Salary: €90,000.",
	}).Body.Close()
	postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Unresolved Corp",
		"jobDescription": "Role with no salary mentioned at all.",
	}).Body.Close()

	resp, err := http.Get(server.URL + "/api/job-listings?sort=ral&order=desc")
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
	if len(results) != 3 {
		t.Fatalf("expected 3 job listings, got %d", len(results))
	}

	companies := make([]string, len(results))
	for i, r := range results {
		companies[i] = r["jobListing"].(map[string]any)["company"].(string)
	}
	want := []string{"High Corp", "Low Corp", "Unresolved Corp"}
	for i := range want {
		if companies[i] != want[i] {
			t.Fatalf("expected order %v, got %v", want, companies)
		}
	}
}

// TestListJobListings_FilterByRALMinMax_ExcludesOutOfRangeAndNonNumeric
// confirms ral_min/ral_max (defaulting currency to EUR) thread through
// GET /api/job-listings end-to-end.
func TestListJobListings_FilterByRALMinMax_ExcludesOutOfRangeAndNonNumeric(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{}
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: client}))
	defer server.Close()

	postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "In Range Corp",
		"jobDescription": "Role. Salary: €45,000.",
	}).Body.Close()
	postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Too High Corp",
		"jobDescription": "Role. Salary: €90,000.",
	}).Body.Close()
	postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Unresolved Corp",
		"jobDescription": "Role with no salary mentioned at all.",
	}).Body.Close()

	resp, err := http.Get(server.URL + "/api/job-listings?ral_min=40000&ral_max=50000")
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
	if len(results) != 1 {
		t.Fatalf("expected 1 job listing within range, got %d: %v", len(results), results)
	}
	if company := results[0]["jobListing"].(map[string]any)["company"]; company != "In Range Corp" {
		t.Errorf("expected In Range Corp, got %v", company)
	}
}

func TestListJobListings_InvalidRALFilterParam_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/job-listings?ral_min=not-a-number")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
