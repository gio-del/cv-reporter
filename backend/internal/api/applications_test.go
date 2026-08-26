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

func patchJSON(t *testing.T, url string, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
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

func saveJobListing(t *testing.T, serverURL, company string) string {
	t.Helper()
	resp := postJSON(t, serverURL+"/api/job-listings", map[string]any{
		"company":        company,
		"jobDescription": "Some role. Salary: €40,000.",
	})
	defer resp.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result["jobListing"].(map[string]any)["id"].(string)
}

func TestUpdateApplicationStatus_AllowedTransition_WritesFileAndReturns200(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp := patchJSON(t, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": "tailoring"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var application map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&application); err != nil {
		t.Fatal(err)
	}
	if application["status"] != "tailoring" {
		t.Errorf("expected status tailoring, got %v", application["status"])
	}

	content, err := os.ReadFile(filepath.Join(dataDir, "applications", id+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("tailoring")) {
		t.Errorf("expected application file to record status tailoring, got:\n%s", content)
	}
}

func TestUpdateApplicationStatus_DisallowedTransition_Returns400AndLeavesFileUnchanged(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp := patchJSON(t, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": "offer"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	content, err := os.ReadFile(filepath.Join(dataDir, "applications", id+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("saved")) {
		t.Errorf("expected application file to still record status saved, got:\n%s", content)
	}
}

func TestUpdateApplicationStatus_UnknownApplication_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	resp := patchJSON(t, server.URL+"/api/applications/does-not-exist/status", map[string]any{"status": "tailoring"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateApplicationStatus_FullHappyPathToOffer(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	for _, status := range []string{"tailoring", "sent", "interviewing", "offer"} {
		resp := patchJSON(t, server.URL+"/api/applications/"+id+"/status", map[string]any{"status": status})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("transitioning to %q: expected 200, got %d", status, resp.StatusCode)
		}
		resp.Body.Close()
	}
}
