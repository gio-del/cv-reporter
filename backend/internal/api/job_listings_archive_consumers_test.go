package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func getStatsBody(t *testing.T, serverURL string) map[string]any {
	t.Helper()
	resp, err := http.Get(serverURL + "/api/applications/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from stats, got %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

// TestApplicationsStats_UnchangedByArchiving is the regression guard issue
// #98 cares most about: archiving is a view concern, so the funnel/stats
// view (#36) must report identical figures before and after.
func TestApplicationsStats_UnchangedByArchiving(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	saveJobListing(t, server.URL, "Saved Corp")
	rejected := saveJobListing(t, server.URL, "Rejected Corp")
	for _, status := range []string{"tailoring", "sent", "interviewing", "rejected"} {
		resp := patchJSON(t, server.URL+"/api/applications/"+rejected+"/status", map[string]any{"status": status})
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("seed transition to %s failed: %d", status, resp.StatusCode)
		}
	}
	savedNeverApplied := saveJobListing(t, server.URL, "Never Applied Corp")

	before := getStatsBody(t, server.URL)
	if before["total"].(float64) != 3 {
		t.Fatalf("expected seed total 3, got %v", before["total"])
	}

	for _, id := range []string{rejected, savedNeverApplied} {
		if status, _ := postArchiveAction(t, server.URL, id, "archive"); status != http.StatusOK {
			t.Fatalf("archive failed: %d", status)
		}
	}

	after := getStatsBody(t, server.URL)
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) {
		t.Errorf("expected stats identical before and after archiving\nbefore: %s\nafter:  %s", beforeJSON, afterJSON)
	}
}

// TestCreateJobListing_DuplicateOfArchivedListing_StillWarns confirms
// duplicate detection (#43) still checks against archived Job Listings, so
// re-saving a role already archived warns instead of silently duplicating.
func TestCreateJobListing_DuplicateOfArchivedListing_StillWarns(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	first := postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Acme Inc.",
		"title":          "Senior Backend Engineer",
		"jobDescription": "Senior Backend Engineer role.",
	})
	var firstResult map[string]any
	if err := json.NewDecoder(first.Body).Decode(&firstResult); err != nil {
		t.Fatal(err)
	}
	first.Body.Close()
	firstID := firstResult["jobListing"].(map[string]any)["id"].(string)
	if status, _ := postArchiveAction(t, server.URL, firstID, "archive"); status != http.StatusOK {
		t.Fatalf("archive failed: %d", status)
	}

	second := postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Acme",
		"title":          "Sr Backend Engineer",
		"jobDescription": "Sr Backend Engineer, captured again.",
	})
	defer second.Body.Close()
	var secondResult map[string]any
	if err := json.NewDecoder(second.Body).Decode(&secondResult); err != nil {
		t.Fatal(err)
	}
	warning, ok := secondResult["duplicateWarning"].(map[string]any)
	if !ok {
		t.Fatalf("expected a duplicateWarning against the archived listing, got %v", secondResult)
	}
	if warning["jobListingId"] != firstID {
		t.Errorf("expected duplicateWarning to point at archived %q, got %v", firstID, warning["jobListingId"])
	}
}

// TestDeleteJobListing_ArchivedListing_StillDeletes confirms Delete (#23)
// is unchanged for an archived Job Listing.
func TestDeleteJobListing_ArchivedListing_StillDeletes(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	postArchiveAction(t, server.URL, id, "archive")

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/job-listings/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	assertCompanies(t, listCompanies(t, server.URL, "?archived=all"), nil)
}
