package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func TestTrackedBoards_AddThenList(t *testing.T) {
	dataDir := seedDataDir(t)
	// Listing tracked boards now computes a live new-count preview per
	// board, so this needs a fake doer rather than a real network call —
	// an omitted RouterConfig.ATSHTTPDoer defaults to http.DefaultClient.
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonATSResponse(http.StatusOK, `{"jobs": []}`), nil
	}}
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}, ATSHTTPDoer: doer}))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/ats/tracked-boards", map[string]any{
		"provider": "greenhouse",
		"slug":     "acme",
		"label":    "Acme Corp",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created["slug"] != "acme" || created["provider"] != "greenhouse" {
		t.Errorf("unexpected created board: %v", created)
	}

	listResp, err := http.Get(server.URL + "/api/ats/tracked-boards")
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.StatusCode)
	}
	var boards []map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&boards); err != nil {
		t.Fatal(err)
	}
	if len(boards) != 1 {
		t.Fatalf("expected 1 tracked board, got %d", len(boards))
	}
}

func TestTrackedBoards_Delete_RemovesIt(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/ats/tracked-boards", map[string]any{"provider": "lever", "slug": "acme"})
	defer resp.Body.Close()
	var created map[string]any
	json.NewDecoder(resp.Body).Decode(&created)
	id := created["id"].(string)

	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/ats/tracked-boards/"+id, nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}

	listResp, err := http.Get(server.URL + "/api/ats/tracked-boards")
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var boards []map[string]any
	json.NewDecoder(listResp.Body).Decode(&boards)
	if len(boards) != 0 {
		t.Errorf("expected no tracked boards after delete, got %d", len(boards))
	}
}

func TestTrackedBoards_Delete_RemovesSeenState(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonATSResponse(http.StatusOK, fixtureGreenhouseResponse), nil
	}}
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}, ATSHTTPDoer: doer}))
	defer server.Close()

	createResp := postJSON(t, server.URL+"/api/ats/tracked-boards", map[string]any{"provider": "greenhouse", "slug": "acme"})
	var created map[string]any
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	id := created["id"].(string)

	fetchResp, err := http.Get(server.URL + "/api/ats/greenhouse/acme/listings")
	if err != nil {
		t.Fatal(err)
	}
	fetchResp.Body.Close()

	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/ats/tracked-boards/"+id, nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()

	// Story 6: re-tracking the same board later should start a fresh
	// baseline, i.e. the very next fetch should seed rather than see a
	// stale seen-set from before it was untracked.
	retrackResp := postJSON(t, server.URL+"/api/ats/tracked-boards", map[string]any{"provider": "greenhouse", "slug": "acme"})
	retrackResp.Body.Close()

	refetchResp, err := http.Get(server.URL + "/api/ats/greenhouse/acme/listings")
	if err != nil {
		t.Fatal(err)
	}
	defer refetchResp.Body.Close()
	var listings []map[string]any
	if err := json.NewDecoder(refetchResp.Body).Decode(&listings); err != nil {
		t.Fatal(err)
	}
	if len(listings) != 1 || listings[0]["new"] != false {
		t.Errorf("expected the re-tracked board's first fetch to seed a fresh baseline (nothing new), got %v", listings)
	}
}

func TestTrackedBoards_List_IncludesNewCount(t *testing.T) {
	dataDir := seedDataDir(t)
	fetchCount := 0
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		fetchCount++
		if fetchCount == 1 {
			return jsonATSResponse(http.StatusOK, fixtureGreenhouseResponse), nil
		}
		return jsonATSResponse(http.StatusOK, `{
		  "jobs": [
		    {"id": 1, "title": "Backend Engineer", "location": {"name": "Remote"}, "absolute_url": "https://boards.greenhouse.io/acme/jobs/1", "content": "<p>Join us.</p>"},
		    {"id": 2, "title": "New Role", "location": {"name": "Remote"}, "absolute_url": "https://boards.greenhouse.io/acme/jobs/2", "content": "<p>New.</p>"}
		  ]
		}`), nil
	}}
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}, ATSHTTPDoer: doer}))
	defer server.Close()

	trackResp := postJSON(t, server.URL+"/api/ats/tracked-boards", map[string]any{"provider": "greenhouse", "slug": "acme"})
	trackResp.Body.Close()

	// Establish a seen baseline (1 listing, none new) via the real fetch
	// path, the only one that mutates seen-state.
	firstFetch, err := http.Get(server.URL + "/api/ats/greenhouse/acme/listings")
	if err != nil {
		t.Fatal(err)
	}
	firstFetch.Body.Close()

	// Reading the tracked-boards list (which computes a count preview)
	// must be read-only: calling it must not itself mark the new listing
	// as seen.
	for i := 0; i < 2; i++ {
		listResp, err := http.Get(server.URL + "/api/ats/tracked-boards")
		if err != nil {
			t.Fatal(err)
		}
		var boards []map[string]any
		if err := json.NewDecoder(listResp.Body).Decode(&boards); err != nil {
			t.Fatal(err)
		}
		listResp.Body.Close()
		if len(boards) != 1 {
			t.Fatalf("expected 1 tracked board, got %d", len(boards))
		}
		newCount, ok := boards[0]["newCount"].(float64)
		if !ok || newCount != 1 {
			t.Errorf("expected newCount=1 (the 2nd, unseen listing), got %v", boards[0]["newCount"])
		}
	}

	// The actual listings fetch should still independently see the new
	// listing as new — the count preview above didn't consume it.
	finalFetch, err := http.Get(server.URL + "/api/ats/greenhouse/acme/listings")
	if err != nil {
		t.Fatal(err)
	}
	defer finalFetch.Body.Close()
	body, _ := io.ReadAll(finalFetch.Body)
	if !strings.Contains(string(body), `"new":true`) {
		t.Errorf("expected the listings fetch to still find the new listing, got %s", body)
	}
}
