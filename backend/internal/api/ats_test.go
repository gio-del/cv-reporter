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

type fakeATSDoer struct {
	do func(*http.Request) (*http.Response, error)
}

func (f fakeATSDoer) Do(req *http.Request) (*http.Response, error) {
	return f.do(req)
}

func jsonATSResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

const fixtureGreenhouseResponse = `{
  "jobs": [
    {
      "id": 1,
      "title": "Backend Engineer",
      "location": {"name": "Remote"},
      "absolute_url": "https://boards.greenhouse.io/acme/jobs/1",
      "content": "<p>Join us.</p>"
    }
  ]
}`

func TestListAtsListings_Greenhouse_ReturnsNormalizedListings(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonATSResponse(http.StatusOK, fixtureGreenhouseResponse), nil
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/ats/greenhouse/acme/listings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var listings []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		t.Fatal(err)
	}
	if len(listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(listings))
	}
	if listings[0]["title"] != "Backend Engineer" {
		t.Errorf("expected title Backend Engineer, got %v", listings[0]["title"])
	}
	if listings[0]["url"] != "https://boards.greenhouse.io/acme/jobs/1" {
		t.Errorf("expected url from absolute_url, got %v", listings[0]["url"])
	}
}

func TestListAtsListings_BoardNotFound_Returns404WithClearError(t *testing.T) {
	dataDir := seedDataDir(t)
	doer := fakeATSDoer{do: func(req *http.Request) (*http.Response, error) {
		return jsonATSResponse(http.StatusNotFound, `{}`), nil
	}}
	server := httptest.NewServer(api.NewRouterWithClients(dataDir, &fakeGenerationClient{}, doer))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/ats/greenhouse/does-not-exist/listings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	got := strings.ToLower(string(body))
	// Story 6: the message must let the user act on it — name the slug they
	// typed and suggest what to try next — not just say "not found".
	if !strings.Contains(got, "does-not-exist") {
		t.Errorf("expected the error to name the slug that wasn't found, got %q", body)
	}
	if !strings.Contains(got, "check the slug") && !strings.Contains(got, "different provider") {
		t.Errorf("expected the error to suggest checking the slug or trying a different provider, got %q", body)
	}
}
