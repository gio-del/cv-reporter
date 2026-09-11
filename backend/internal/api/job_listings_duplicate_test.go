package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func TestCreateJobListing_LikelyDuplicate_SavesAnywayAndWarns(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()

	first := postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Acme Inc.",
		"jobDescription": "Senior Backend Engineer role.",
		"title":          "Senior Backend Engineer",
	})
	defer first.Body.Close()
	var firstResult map[string]any
	if err := json.NewDecoder(first.Body).Decode(&firstResult); err != nil {
		t.Fatal(err)
	}
	firstID := firstResult["jobListing"].(map[string]any)["id"].(string)

	second := postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Acme",
		"jobDescription": "Sr Backend Engineer, captured from a different source.",
		"title":          "Sr Backend Engineer",
	})
	defer second.Body.Close()
	if second.StatusCode != http.StatusCreated {
		t.Fatalf("expected the near-duplicate to still save with 201, got %d", second.StatusCode)
	}

	var secondResult map[string]any
	if err := json.NewDecoder(second.Body).Decode(&secondResult); err != nil {
		t.Fatal(err)
	}
	secondID := secondResult["jobListing"].(map[string]any)["id"].(string)
	if secondID == firstID {
		t.Fatalf("expected the near-duplicate to be persisted as a distinct listing, got the same id %q", secondID)
	}

	warning, ok := secondResult["duplicateWarning"].(map[string]any)
	if !ok {
		t.Fatalf("expected a duplicateWarning on the response, got %v", secondResult)
	}
	if warning["jobListingId"] != firstID {
		t.Errorf("expected duplicateWarning to point at %q, got %v", firstID, warning["jobListingId"])
	}
	if warning["company"] != "Acme Inc." {
		t.Errorf("expected duplicateWarning to carry the matched listing's company, got %v", warning["company"])
	}
	if _, ok := warning["score"].(float64); !ok {
		t.Errorf("expected duplicateWarning to carry a numeric score, got %v", warning["score"])
	}
}

func TestCreateJobListing_NoSimilarExistingListing_NoWarning(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Globex Corp",
		"jobDescription": "A totally unrelated Product Designer role.",
		"title":          "Product Designer",
	})
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if _, present := result["duplicateWarning"]; present {
		t.Errorf("expected no duplicateWarning with no prior listings, got %v", result["duplicateWarning"])
	}
}
