package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func putJSON(t *testing.T, url string, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestUpdateEntry_ValidPayload_WritesFileAndReturns200(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	payload := map[string]any{
		"employer": "Quantyca S.p.A.",
		"role":     "Senior Data Engineer",
		"client":   "Amplifon",
		"location": "Monza",
		"start":    "2024-10",
		"end":      nil,
		"flagship": true,
		"tags":     []string{"AI Platform", "Go"},
		"bullets": []string{
			"Led the AI Platform rebuild.",
			"Mentored two engineers.",
		},
	}

	resp := putJSON(t, server.URL+"/api/master-data/entries/experience/quantyca-amplifon", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := os.ReadFile(filepath.Join(dataDir, "experience", "quantyca-amplifon.md"))
		t.Fatalf("expected 200, got %d; file on disk:\n%s", resp.StatusCode, body)
	}

	var entry map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		t.Fatal(err)
	}
	if entry["role"] != "Senior Data Engineer" {
		t.Errorf("expected response role Senior Data Engineer, got %v", entry["role"])
	}

	updated, err := os.ReadFile(filepath.Join(dataDir, "experience", "quantyca-amplifon.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(updated)
	if !bytes.Contains(updated, []byte("Senior Data Engineer")) {
		t.Errorf("expected file to contain updated role, got:\n%s", content)
	}
	if !bytes.Contains(updated, []byte("Led the AI Platform rebuild.")) {
		t.Errorf("expected file to contain updated bullet, got:\n%s", content)
	}
	if bytes.Contains(updated, []byte("Designed and built an AI Platform.")) {
		t.Errorf("expected old bullet to be gone, got:\n%s", content)
	}
}

func TestUpdateEntry_MalformedDate_Returns400AndFileUnchanged(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	original, err := os.ReadFile(filepath.Join(dataDir, "experience", "quantyca-amplifon.md"))
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"employer": "Quantyca S.p.A.",
		"role":     "Data Engineer",
		"client":   "Amplifon",
		"location": "Monza",
		"start":    "not-a-date",
		"tags":     []string{"AI Platform"},
		"bullets":  []string{"Did stuff."},
	}

	resp := putJSON(t, server.URL+"/api/master-data/entries/experience/quantyca-amplifon", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	body, _ := os.ReadFile(filepath.Join(dataDir, "experience", "quantyca-amplifon.md"))
	if !bytes.Equal(body, original) {
		t.Errorf("expected file to be untouched after invalid update, got:\n%s", body)
	}
}

func TestUpdateEntry_MissingRequiredField_Returns400AndFileUnchanged(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	original, err := os.ReadFile(filepath.Join(dataDir, "experience", "quantyca-amplifon.md"))
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"employer": "",
		"role":     "Data Engineer",
		"client":   "Amplifon",
		"location": "Monza",
		"start":    "2024-10",
		"tags":     []string{"AI Platform"},
		"bullets":  []string{"Did stuff."},
	}

	resp := putJSON(t, server.URL+"/api/master-data/entries/experience/quantyca-amplifon", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	body, _ := os.ReadFile(filepath.Join(dataDir, "experience", "quantyca-amplifon.md"))
	if !bytes.Equal(body, original) {
		t.Errorf("expected file to be untouched after invalid update, got:\n%s", body)
	}
}

func TestUpdateEntry_UnknownID_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	payload := map[string]any{
		"employer": "Somewhere",
		"role":     "Role",
		"start":    "2024-10",
		"tags":     []string{},
		"bullets":  []string{},
	}

	resp := putJSON(t, server.URL+"/api/master-data/entries/experience/does-not-exist", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
