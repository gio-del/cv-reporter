package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// The tests in this file cover RouterConfig's defaulting rules — the only
// genuinely new logic introduced by collapsing the seven NewRouter*
// constructors into one. Every route's own behavior is covered by the rest
// of the package's tests; these assert only that omitting a field lands on
// the documented default, through the same external HTTP surface.

// TestRouterConfig_OmittedProjectRoot_MatchesExplicitDot pins the
// ProjectRoot default: "" means ".", the directory Render passes to typst
// as its single --root (ADR-0012). Exercised through the project-root-
// backed generation-file route rather than a live render, so the
// equivalence is asserted on served bytes instead of on a typst run.
func TestRouterConfig_OmittedProjectRoot_MatchesExplicitDot(t *testing.T) {
	projectRoot, dataDir := seedProjectRoot(t)
	outputDir := filepath.Join(projectRoot, "output", "acme-corp")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pdfBytes := []byte("%PDF-1.7 fake pdf content")
	if err := os.WriteFile(filepath.Join(outputDir, "cv.pdf"), pdfBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	// "." is only meaningful relative to the process working directory, so
	// make the seeded project root the working directory for this test.
	t.Chdir(projectRoot)

	omitted := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, GenerationClient: &fakeGenerationClient{}}))
	defer omitted.Close()
	explicitDot := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir, ProjectRoot: ".", GenerationClient: &fakeGenerationClient{}}))
	defer explicitDot.Close()

	for _, tc := range []struct {
		name   string
		server *httptest.Server
	}{
		{"omitted project root", omitted},
		{"explicit dot", explicitDot},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.server.URL + "/api/generations/acme-corp/cv.pdf")
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
			if string(body) != string(pdfBytes) {
				t.Errorf("expected the file under the project root to be served, got %q", body)
			}
		})
	}
}

// TestRouterConfig_OmittedATSHTTPDoer_ServesBoardRoutesThatNeverFetch pins
// the ATSHTTPDoer default: nil means http.DefaultClient. Add-then-delete of
// a tracked board never issues a board request, so this exercises the ATS
// routes with the default doer in place without reaching a real
// Greenhouse/Lever/Ashby board.
func TestRouterConfig_OmittedATSHTTPDoer_ServesBoardRoutesThatNeverFetch(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/ats/tracked-boards", map[string]any{"provider": "lever", "slug": "acme"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/ats/tracked-boards/"+created["id"].(string), nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}
}

// TestRouterConfig_EmptyDataDir_Panics pins DataDir as the one required
// field: a router built without it would read an empty Master Data set
// rooted at the process working directory and look like data loss, so a
// mis-wired construction has to fail loudly and immediately instead.
func TestRouterConfig_EmptyDataDir_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected NewRouter to panic on an empty DataDir")
		}
	}()

	api.NewRouter(api.RouterConfig{})
}

// TestRouterConfig_OmittedLANAuthToken_LeavesRoutesUnauthenticated pins the
// LANAuthToken default: omitting it wires in no check at all, preserving
// ADR-0004's localhost-only, no-auth default. Its counterpart — a
// configured token rejecting a request without the header — lives in
// lan_auth_integration_test.go.
func TestRouterConfig_OmittedLANAuthToken_LeavesRoutesUnauthenticated(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with no token header, got %d", resp.StatusCode)
	}
}
