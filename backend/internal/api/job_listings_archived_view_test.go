package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// listCompanies GETs /api/job-listings with query (which may be empty) and
// returns each row's company in response order.
func listCompanies(t *testing.T, serverURL, query string) []string {
	t.Helper()
	resp, err := http.Get(serverURL + "/api/job-listings" + query)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/job-listings%s: expected 200, got %d", query, resp.StatusCode)
	}
	var rows []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	companies := make([]string, len(rows))
	for i, row := range rows {
		companies[i] = row["jobListing"].(map[string]any)["company"].(string)
	}
	return companies
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func assertCompanies(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(sorted(got), sorted(want)) {
		t.Errorf("expected companies %v, got %v", want, got)
	}
}

// seedActiveAndArchived saves Active Corp and Archived Corp, archiving the
// latter over the API.
func seedActiveAndArchived(t *testing.T, serverURL string) (activeID, archivedID string) {
	t.Helper()
	activeID = saveJobListing(t, serverURL, "Active Corp")
	archivedID = saveJobListing(t, serverURL, "Archived Corp")
	if status, _ := postArchiveAction(t, serverURL, archivedID, "archive"); status != http.StatusOK {
		t.Fatalf("archive seed failed: %d", status)
	}
	return activeID, archivedID
}

func TestListJobListings_Default_OmitsArchived(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()
	seedActiveAndArchived(t, server.URL)

	assertCompanies(t, listCompanies(t, server.URL, ""), []string{"Active Corp"})
	assertCompanies(t, listCompanies(t, server.URL, "?archived=exclude"), []string{"Active Corp"})
}

func TestListJobListings_ArchivedOnly_ReturnsOnlyArchived(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()
	seedActiveAndArchived(t, server.URL)

	assertCompanies(t, listCompanies(t, server.URL, "?archived=only"), []string{"Archived Corp"})
}

func TestListJobListings_ArchivedAll_ReturnsBoth(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()
	seedActiveAndArchived(t, server.URL)

	assertCompanies(t, listCompanies(t, server.URL, "?archived=all"), []string{"Active Corp", "Archived Corp"})
}

func TestListJobListings_InvalidArchivedValue_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/job-listings?archived=true")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for archived=true, got %d", resp.StatusCode)
	}
}

func TestListJobListings_RowsCarryArchivedFlag(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()
	seedActiveAndArchived(t, server.URL)

	resp, err := http.Get(server.URL + "/api/job-listings?archived=all")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var rows []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		listing := row["jobListing"].(map[string]any)
		want := listing["company"] == "Archived Corp"
		if listing["archived"] != want {
			t.Errorf("%v: expected archived %v, got %v", listing["company"], want, listing["archived"])
		}
	}
}

func TestListJobListings_UnarchiveReturnsListingToDefaultView(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()
	_, archivedID := seedActiveAndArchived(t, server.URL)

	if status, _ := postArchiveAction(t, server.URL, archivedID, "unarchive"); status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}

	assertCompanies(t, listCompanies(t, server.URL, ""), []string{"Active Corp", "Archived Corp"})
	assertCompanies(t, listCompanies(t, server.URL, "?archived=only"), nil)
}

func TestListJobListings_FileWithoutArchivedKey_AppearsInDefaultView(t *testing.T) {
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
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()

	assertCompanies(t, listCompanies(t, server.URL, ""), []string{"Legacy Corp"})
}

func TestListJobListings_ArchivedView_ComposesWithStatusFilter(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()

	savedArchived := saveJobListing(t, server.URL, "Saved Archived Corp")
	tailoringArchived := saveJobListing(t, server.URL, "Tailoring Archived Corp")
	tailoringActive := saveJobListing(t, server.URL, "Tailoring Active Corp")
	for _, id := range []string{tailoringArchived, tailoringActive} {
		patchJSON(t, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": "tailoring"}).Body.Close()
	}
	for _, id := range []string{savedArchived, tailoringArchived} {
		postArchiveAction(t, server.URL, id, "archive")
	}

	assertCompanies(t, listCompanies(t, server.URL, "?status=tailoring"), []string{"Tailoring Active Corp"})
	assertCompanies(t, listCompanies(t, server.URL, "?status=tailoring&archived=only"), []string{"Tailoring Archived Corp"})
	assertCompanies(t, listCompanies(t, server.URL, "?status=tailoring&archived=all"), []string{"Tailoring Active Corp", "Tailoring Archived Corp"})
}

func TestListJobListings_ArchivedView_ComposesWithRALFilterAndSort(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()

	save := func(company, jd string) string {
		resp := postJSON(t, server.URL+"/api/job-listings", map[string]any{"company": company, "jobDescription": jd})
		defer resp.Body.Close()
		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		return result["jobListing"].(map[string]any)["id"].(string)
	}
	save("Low Active Corp", "Role. Salary: €40,000.")
	lowArchived := save("Low Archived Corp", "Role. Salary: €45,000.")
	highArchived := save("High Archived Corp", "Role. Salary: €90,000.")
	save("High Active Corp", "Role. Salary: €95,000.")
	for _, id := range []string{lowArchived, highArchived} {
		postArchiveAction(t, server.URL, id, "archive")
	}

	got := listCompanies(t, server.URL, "?archived=only&sort=ral&order=desc")
	if want := []string{"High Archived Corp", "Low Archived Corp"}; !reflect.DeepEqual(got, want) {
		t.Errorf("expected archived-only RAL sort %v, got %v", want, got)
	}
	assertCompanies(t, listCompanies(t, server.URL, "?archived=only&ral_min=30000&ral_max=50000"), []string{"Low Archived Corp"})
	assertCompanies(t, listCompanies(t, server.URL, "?ral_min=30000&ral_max=50000"), []string{"Low Active Corp"})
}
