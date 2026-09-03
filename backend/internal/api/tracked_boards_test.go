package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func TestTrackedBoards_AddThenList(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
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
	server := httptest.NewServer(api.NewRouter(dataDir))
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
