package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// The ordering matrix for GET /api/applications lives with
// tracking.GroupApplications; these tests only confirm the route is wired,
// the envelope's shape, the archived view and error propagation (issue #95).

type applicationsViewResponse struct {
	Total  int `json:"total"`
	Groups []struct {
		Status string           `json:"status"`
		Count  int              `json:"count"`
		Items  []map[string]any `json:"items"`
	} `json:"groups"`
}

func getApplicationsView(t *testing.T, serverURL, query string) (int, applicationsViewResponse) {
	t.Helper()
	resp, err := http.Get(serverURL + "/api/applications" + query)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body applicationsViewResponse
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode, body
}

func viewCompanies(body applicationsViewResponse) []string {
	var companies []string
	for _, g := range body.Groups {
		for _, item := range g.Items {
			companies = append(companies, item["jobListing"].(map[string]any)["company"].(string))
		}
	}
	return companies
}

func TestListApplications_ReturnsGroupedEnvelopeWithoutJobDescription(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	saveJobListing(t, server.URL, "Acme Corp")
	sentID := saveJobListing(t, server.URL, "Globex")
	for _, status := range []string{"tailoring", "sent"} {
		resp := patchJSON(t, server.URL+"/api/applications/"+sentID+"/status", map[string]any{"status": status})
		resp.Body.Close()
	}

	status, body := getApplicationsView(t, server.URL, "")
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}

	wantOrder := []string{"saved", "tailoring", "sent", "interviewing", "offer", "rejected", "withdrawn"}
	if len(body.Groups) != len(wantOrder) {
		t.Fatalf("expected %d groups, got %d", len(wantOrder), len(body.Groups))
	}
	for i, g := range body.Groups {
		if g.Status != wantOrder[i] {
			t.Errorf("group %d: expected %q, got %q", i, wantOrder[i], g.Status)
		}
		if g.Items == nil {
			t.Errorf("group %q: expected an items array, even when empty", g.Status)
		}
	}
	if body.Total != 2 {
		t.Errorf("expected total 2, got %d", body.Total)
	}

	saved, sent := body.Groups[0], body.Groups[2]
	if saved.Count != 1 || sent.Count != 1 {
		t.Fatalf("expected one Application each in Saved and Sent, got %d and %d", saved.Count, sent.Count)
	}

	item := sent.Items[0]
	listing := item["jobListing"].(map[string]any)
	if listing["company"] != "Globex" {
		t.Errorf("expected Globex in Sent, got %v", listing["company"])
	}
	if _, present := listing["jobDescription"]; present {
		t.Errorf("expected no jobDescription in an Applications view item, got %v", listing["jobDescription"])
	}
	if listing["hasJobDescription"] != true {
		t.Errorf("expected hasJobDescription true, got %v", listing["hasJobDescription"])
	}
	application := item["application"].(map[string]any)
	if application["status"] != "sent" {
		t.Errorf("expected the paired Application in Status sent, got %v", application["status"])
	}
	if _, present := application["isStale"]; !present {
		t.Errorf("expected isStale on the Application")
	}
}

func TestListApplications_ArchivedJobListingsExcludedByDefault_IncludedOnRequest(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	seedActiveAndArchived(t, server.URL)

	cases := []struct {
		query string
		want  []string
	}{
		{"", []string{"Active Corp"}},
		{"?archived=exclude", []string{"Active Corp"}},
		{"?archived=only", []string{"Archived Corp"}},
		{"?archived=all", []string{"Active Corp", "Archived Corp"}},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			status, body := getApplicationsView(t, server.URL, c.query)
			if status != http.StatusOK {
				t.Fatalf("expected 200, got %d", status)
			}
			assertCompanies(t, viewCompanies(body), c.want)
			if body.Total != len(c.want) {
				t.Errorf("expected total %d, got %d", len(c.want), body.Total)
			}
		})
	}
}

func TestListApplications_InvalidArchivedParam_Returns400(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	if status, _ := getApplicationsView(t, server.URL, "?archived=sometimes"); status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", status)
	}
}

func TestListApplications_ReadFailure_Returns500(t *testing.T) {
	dataDir := seedDataDir(t)
	writeFile(t, filepath.Join(dataDir, "jobs", "broken.md"), "no frontmatter here")
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	if status, _ := getApplicationsView(t, server.URL, ""); status != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", status)
	}
}
