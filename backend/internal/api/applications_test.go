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

func TestUpdateApplicationMethod_ValidCorrection_WritesFileAndReturns200(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp := patchJSON(t, server.URL+"/api/applications/"+id+"/method", map[string]any{
		"kind":  "portal",
		"value": "https://acme.example/apply",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var application map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&application); err != nil {
		t.Fatal(err)
	}
	method := application["method"].(map[string]any)
	if method["kind"] != "portal" || method["value"] != "https://acme.example/apply" {
		t.Errorf("expected corrected method, got %v", method)
	}

	content, err := os.ReadFile(filepath.Join(dataDir, "applications", id+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("portal")) {
		t.Errorf("expected application file to record the corrected method, got:\n%s", content)
	}
}

func TestUpdateApplicationMethod_UnknownKind_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp := patchJSON(t, server.URL+"/api/applications/"+id+"/method", map[string]any{"kind": "carrier-pigeon"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateApplicationMethod_UnknownApplication_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	resp := patchJSON(t, server.URL+"/api/applications/does-not-exist/method", map[string]any{"kind": "portal"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRecordApplicationGeneration_AppendsHistoryAcrossRegenerates(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp1 := postJSON(t, server.URL+"/api/applications/"+id+"/generations", map[string]any{
		"slug":   "acme-corp",
		"cvPath": "output/acme-corp/cv.pdf",
	})
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp1.StatusCode)
	}

	resp2 := postJSON(t, server.URL+"/api/applications/"+id+"/generations", map[string]any{
		"slug":            "acme-corp",
		"cvPath":          "output/acme-corp/cv.pdf",
		"coverLetterPath": "output/acme-corp/cover-letter.pdf",
	})
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp2.StatusCode)
	}

	var application map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&application); err != nil {
		t.Fatal(err)
	}
	generations, ok := application["generations"].([]any)
	if !ok || len(generations) != 2 {
		t.Fatalf("expected regenerating to append rather than overwrite (2 records), got %v", application["generations"])
	}
	latest := generations[1].(map[string]any)
	if latest["coverLetterPath"] != "output/acme-corp/cover-letter.pdf" {
		t.Errorf("expected latest record's coverLetterPath, got %v", latest)
	}
}

func TestRecordApplicationGeneration_MissingSlug_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp := postJSON(t, server.URL+"/api/applications/"+id+"/generations", map[string]any{"cvPath": "output/x/cv.pdf"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRecordApplicationGeneration_UnknownApplication_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/applications/does-not-exist/generations", map[string]any{
		"slug":   "acme-corp",
		"cvPath": "output/acme-corp/cv.pdf",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateApplicationContact_ValidPayload_WritesFileAndReturns200(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp := patchJSON(t, server.URL+"/api/applications/"+id+"/contact", map[string]any{
		"name":  "Jane Recruiter",
		"email": "jane@acme.example",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var application map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&application); err != nil {
		t.Fatal(err)
	}
	contact := application["contact"].(map[string]any)
	if contact["name"] != "Jane Recruiter" || contact["email"] != "jane@acme.example" {
		t.Errorf("expected saved contact, got %v", contact)
	}

	content, err := os.ReadFile(filepath.Join(dataDir, "applications", id+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("jane@acme.example")) {
		t.Errorf("expected application file to record the contact, got:\n%s", content)
	}
}

func TestUpdateApplicationContact_MissingEmail_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp := patchJSON(t, server.URL+"/api/applications/"+id+"/contact", map[string]any{"name": "Jane Recruiter"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUpdateApplicationContact_UnknownApplication_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	resp := patchJSON(t, server.URL+"/api/applications/does-not-exist/contact", map[string]any{
		"name":  "Jane Recruiter",
		"email": "jane@acme.example",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
