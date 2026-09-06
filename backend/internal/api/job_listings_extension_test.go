package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

func TestCaptureJobListingFromExtension_ValidPayload_WritesFilesAndCreatesSavedApplication(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{
		"title":       "Backend Engineer",
		"company":     "Acme Corp",
		"location":    "Milan, Italy",
		"url":         "https://www.linkedin.com/jobs/view/12345",
		"description": "Go backend engineer, remote friendly.",
	}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	listing, ok := result["jobListing"].(map[string]any)
	if !ok {
		t.Fatalf("expected a jobListing object, got %v", result["jobListing"])
	}
	id, ok := listing["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected a generated jobListing id, got %v", listing["id"])
	}
	if listing["company"] != "Acme Corp" {
		t.Errorf("expected company Acme Corp, got %v", listing["company"])
	}
	if listing["url"] != "https://www.linkedin.com/jobs/view/12345" {
		t.Errorf("expected url to carry through, got %v", listing["url"])
	}

	application, ok := result["application"].(map[string]any)
	if !ok {
		t.Fatalf("expected an application object, got %v", result["application"])
	}
	if application["jobListingId"] != id {
		t.Errorf("expected application jobListingId %q, got %v", id, application["jobListingId"])
	}
	if application["status"] != "saved" {
		t.Errorf("expected application status saved, got %v", application["status"])
	}

	jobFile, err := os.ReadFile(filepath.Join(dataDir, "jobs", id+".md"))
	if err != nil {
		t.Fatalf("expected job listing file to exist: %v", err)
	}
	if !strings.Contains(string(jobFile), "Go backend engineer, remote friendly.") {
		t.Errorf("expected job listing file to contain the captured description, got:\n%s", jobFile)
	}
}

func TestCaptureJobListingFromExtension_TitleCarriesThrough(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{
		"title":       "Backend Engineer",
		"company":     "Acme Corp",
		"description": "Go backend engineer, remote friendly.",
	}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	if listing["title"] != "Backend Engineer" {
		t.Errorf("expected title to carry through, got %v", listing["title"])
	}
}

// A minimal valid 1x1 PNG, so the downloaded bytes sniff as image/png.
var fixturePNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func TestCaptureJobListingFromExtension_LogoURLPresent_DownloadsAndPersistsLogo(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://media.linkedin.com/logo.png" {
			t.Errorf("expected a request to the logo URL, got %s", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(bytes.NewReader(fixturePNG)),
		}, nil
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	payload := map[string]any{
		"company":     "Acme Corp",
		"description": "Go backend engineer, remote friendly.",
		"logoUrl":     "https://media.linkedin.com/logo.png",
	}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	logo, ok := listing["logo"].(string)
	if !ok || logo == "" {
		t.Fatalf("expected a non-empty logo filename, got %v", listing["logo"])
	}

	logoBytes, err := os.ReadFile(filepath.Join(dataDir, "jobs", logo))
	if err != nil {
		t.Fatalf("expected the downloaded logo file to exist on disk: %v", err)
	}
	if !bytes.Equal(logoBytes, fixturePNG) {
		t.Errorf("expected the downloaded logo file to contain the fetched bytes")
	}
}

func TestGetJobListingLogo_ServesDownloadedLogo(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(bytes.NewReader(fixturePNG)),
		}, nil
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	saveResp := postJSON(t, server.URL+"/api/job-listings/from-extension", map[string]any{
		"company":     "Acme Corp",
		"description": "Go backend engineer.",
		"logoUrl":     "https://media.linkedin.com/logo.png",
	})
	var saved map[string]any
	json.NewDecoder(saveResp.Body).Decode(&saved)
	saveResp.Body.Close()
	id := saved["jobListing"].(map[string]any)["id"].(string)

	resp, err := http.Get(server.URL + "/api/job-listings/" + id + "/logo")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, fixturePNG) {
		t.Errorf("expected the served bytes to match the downloaded logo")
	}
}

func TestGetJobListingLogo_NoLogo_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	resp, err := http.Get(server.URL + "/api/job-listings/" + id + "/logo")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestCaptureJobListingFromExtension_LogoURLNotAnImage_StillSavesWithoutLogo(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
			Body:       io.NopCloser(strings.NewReader("not an image")),
		}, nil
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	payload := map[string]any{
		"company":     "Acme Corp",
		"description": "Go backend engineer, remote friendly.",
		"logoUrl":     "https://media.linkedin.com/not-a-logo.txt",
	}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	if logo, present := listing["logo"]; present {
		t.Errorf("expected no logo field when the downloaded content isn't an image, got %v", logo)
	}

	id := listing["id"].(string)
	entries, err := os.ReadDir(filepath.Join(dataDir, "jobs"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != id+".md" {
			t.Errorf("expected no stray non-image file to be written, found %s", e.Name())
		}
	}
}

func TestCaptureJobListingFromExtension_LogoDownloadFails_StillSavesWithoutLogo(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	payload := map[string]any{
		"company":     "Acme Corp",
		"description": "Go backend engineer, remote friendly.",
		"logoUrl":     "https://media.linkedin.com/logo.png",
	}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	if logo, present := listing["logo"]; present {
		t.Errorf("expected no logo field when the download fails, got %v", logo)
	}
}

func TestCaptureJobListingFromExtension_NoLogoURL_SavesWithoutLogo(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp", "description": "Go backend engineer, remote friendly."}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	if logo, present := listing["logo"]; present {
		t.Errorf("expected no logo field when no logoUrl was captured, got %v", logo)
	}
}

// Confirms the extension endpoint inherits Save's best-effort resolution
// (PRD "Resilient Job Listing Save", story 14) without re-testing every
// RAL/Method failure combination the shared Save path already covers.
func TestCaptureJobListingFromExtension_MethodInferenceFails_StillSavesWithUnresolvedMethod(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{
		inferApplicationMethod: func(ctx context.Context, jobDescription string) (tracking.ApplicationMethod, error) {
			return tracking.ApplicationMethod{}, errors.New("claude api unreachable")
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp", "description": "Go backend engineer, remote friendly."}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	application := result["application"].(map[string]any)
	method := application["method"].(map[string]any)
	if method["kind"] != "unresolved" {
		t.Errorf("expected unresolved Application Method, got %v", method)
	}
}

func TestCaptureJobListingFromExtension_MissingCompany_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"description": "Go backend engineer, remote friendly."}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCaptureJobListingFromExtension_MissingDescription_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp"}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// The caller is a browser extension's background script running on a
// moz-extension:// or chrome-extension:// origin — the browser CORS-checks
// that cross-origin fetch regardless of the extension's declared
// host_permissions on at least some Firefox versions (observed in manual
// verification), so this endpoint answers with permissive CORS headers
// rather than depending on that exemption holding.
func TestCaptureJobListingFromExtension_PostResponseIncludesCORSHeader(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	payload := map[string]any{"company": "Acme Corp", "description": "Go backend engineer."}
	resp := postJSON(t, server.URL+"/api/job-listings/from-extension", payload)
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
}

func TestCaptureJobListingFromExtension_OptionsPreflight_Returns204WithCORSHeaders(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodOptions, server.URL+"/api/job-listings/from-extension", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Errorf("expected Access-Control-Allow-Methods to include POST, got %q", got)
	}
}
