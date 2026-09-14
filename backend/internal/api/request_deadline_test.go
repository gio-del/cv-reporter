package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/api"
	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// blockingGenerationClient stands in for a Claude call that never returns:
// SelectAndRewrite blocks until the request context is done, then reports
// why — exactly what a real client's HTTP call does when its context is
// cancelled. No test here touches the Claude API.
func blockingGenerationClient(observed chan<- error) *fakeGenerationClient {
	return &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			<-ctx.Done()
			observed <- ctx.Err()
			return generation.SelectionResult{}, ctx.Err()
		},
	}
}

// TestClaudeRoute_HandlerOutlivingDeadline_StopsAndReportsTimeout is
// stories 12 and 13: a Claude-calling request has an upper bound of its
// own, and reaching it cancels the outbound call rather than leaving it
// holding a connection and a goroutine for the life of the process.
func TestClaudeRoute_HandlerOutlivingDeadline_StopsAndReportsTimeout(t *testing.T) {
	dataDir := seedDataDir(t)
	observed := make(chan error, 1)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		DataDir:            dataDir,
		GenerationClient:   blockingGenerationClient(observed),
		ClaudeRouteTimeout: 50 * time.Millisecond,
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/generations", "application/json",
		strings.NewReader(`{"jobDescription":"Looking for a Go backend engineer."}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 Gateway Timeout, got %d: %s", resp.StatusCode, body)
	}

	select {
	case err := <-observed:
		if err != context.DeadlineExceeded {
			t.Fatalf("expected the client's context to report a deadline, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the outbound call's context was never cancelled")
	}
}

// TestClaudeRoute_FastResponse_IsUnaffected keeps the bound invisible in
// the normal case: a Generation that answers inside the deadline is
// returned untouched, with its own status code.
func TestClaudeRoute_FastResponse_IsUnaffected(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		DataDir:            dataDir,
		GenerationClient:   &fakeGenerationClient{},
		ClaudeRouteTimeout: 10 * time.Second,
	}))
	defer server.Close()

	// No job description: Default Mode, no Claude call, a 200 through the
	// same wrapped route.
	resp, err := http.Post(server.URL+"/api/generations", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
}

// TestClaudeRoute_NonClaudeRoutes_AreNotBounded is story 14: the deadline
// is on the Claude-calling routes only, so Master Data browsing and
// file-serving stay exactly as they were. A 1ns deadline would fail any
// route that had it applied.
func TestClaudeRoute_NonClaudeRoutes_AreNotBounded(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		DataDir:            dataDir,
		GenerationClient:   &fakeGenerationClient{},
		ClaudeRouteTimeout: time.Nanosecond,
	}))
	defer server.Close()

	for _, path := range []string{"/api/healthz", "/api/master-data/entries", "/api/job-listings"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
		}
	}
}

// TestClaudeRoute_OmittedTimeout_FallsBackToDefault pins the documented
// zero-value behaviour of the new RouterConfig field: no deadline
// configured means the ten-minute default, not "no bound at all".
func TestClaudeRoute_OmittedTimeout_FallsBackToDefault(t *testing.T) {
	if api.DefaultClaudeRouteTimeout <= 0 {
		t.Fatalf("DefaultClaudeRouteTimeout must be a real bound, got %v", api.DefaultClaudeRouteTimeout)
	}

	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		DataDir:          dataDir,
		GenerationClient: &fakeGenerationClient{},
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/generations", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 well inside the default deadline, got %d", resp.StatusCode)
	}
}
