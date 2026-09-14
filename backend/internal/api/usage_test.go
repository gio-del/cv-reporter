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
	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

func TestGetUsage_SumsGenerationAndStandaloneUsage(t *testing.T) {
	dataDir := seedDataDir(t)

	// One Job Listing save records standalone RAL-estimation usage.
	client := &fakeGenerationClientWithUsage{
		usage: []generation.CallUsage{
			{CallType: "ral_estimation", InputTokens: 10, OutputTokens: 5, EstimatedCostUSD: 0.001},
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")

	// One recorded Generation carries its own per-call usage breakdown.
	resp := postJSON(t, server.URL+"/api/applications/"+id+"/generations", map[string]any{
		"slug":   "acme-corp",
		"cvPath": "output/acme-corp/cv.pdf",
		"usage": map[string]any{
			"inputTokens":      1000,
			"outputTokens":     200,
			"estimatedCostUsd": 0.012,
			"calls": []map[string]any{
				{"callType": "selection_rewrite", "inputTokens": 1000, "outputTokens": 200, "estimatedCostUsd": 0.012},
			},
		},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 recording the generation, got %d", resp.StatusCode)
	}

	usageResp, err := http.Get(server.URL + "/api/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer usageResp.Body.Close()
	if usageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", usageResp.StatusCode)
	}

	var got map[string]any
	if err := json.NewDecoder(usageResp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if cost, ok := got["estimatedCostUsd"].(float64); !ok || cost < 0.0129 || cost > 0.0131 {
		t.Errorf("estimatedCostUsd = %v, want ~0.013 (0.001 standalone + 0.012 generation)", got["estimatedCostUsd"])
	}
	calls, ok := got["calls"].([]any)
	if !ok || len(calls) != 2 {
		t.Fatalf("expected 2 calls in the breakdown (1 standalone + 1 generation), got %v", got["calls"])
	}
	assertUsageComplete(t, got)
}

func TestGetUsage_NothingRecordedYet_ReturnsZeroValue(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	got := getUsage(t, server.URL)
	if got["estimatedCostUsd"] != float64(0) {
		t.Errorf("estimatedCostUsd = %v, want 0", got["estimatedCostUsd"])
	}
	assertUsageComplete(t, got)
}

func TestGetUsage_CorruptUsageLog_ReportsIncomplete(t *testing.T) {
	dataDir := seedDataDir(t)
	if err := os.WriteFile(filepath.Join(dataDir, "usage-log.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	assertUsageIncomplete(t, getUsage(t, server.URL))
}

// getUsage fetches GET /api/usage and decodes its body, failing on non-200.
func getUsage(t *testing.T, serverURL string) map[string]any {
	t.Helper()
	resp, err := http.Get(serverURL + "/api/usage")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /api/usage, got %d", resp.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

func assertUsageComplete(t *testing.T, usage map[string]any) {
	t.Helper()
	if usage["incomplete"] == true || usage["incompleteReason"] != nil {
		t.Errorf("expected a complete usage total with no incompleteness signal, got incomplete=%v incompleteReason=%v",
			usage["incomplete"], usage["incompleteReason"])
	}
}

func assertUsageIncomplete(t *testing.T, usage map[string]any) {
	t.Helper()
	if usage["incomplete"] != true {
		t.Errorf("incomplete = %v, want true", usage["incomplete"])
	}
	if reason, _ := usage["incompleteReason"].(string); reason == "" {
		t.Errorf("incompleteReason = %v, want a non-empty explanation", usage["incompleteReason"])
	}
}

// usageReportingClient returns a fake generation client whose next drain
// reports one standalone RAL-estimation call, as a Job Listing save does.
func usageReportingClient() *fakeGenerationClientWithUsage {
	return &fakeGenerationClientWithUsage{
		usage: []generation.CallUsage{
			{CallType: "ral_estimation", InputTokens: 10, OutputTokens: 5, EstimatedCostUSD: 0.001},
		},
	}
}

func TestSaveJobListing_CorruptUsageLog_LeavesLogUntouched(t *testing.T) {
	dataDir := seedDataDir(t)
	logPath := filepath.Join(dataDir, "usage-log.json")
	corrupt := []byte(`[{"callType": "ral_estimation", "estimatedCostUsd": 0.5}, {"callType": tru`)
	if err := os.WriteFile(logPath, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, usageReportingClient()))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Acme Corp",
		"jobDescription": "Some role.",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected the save to still succeed with 201, got %d", resp.StatusCode)
	}

	after, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, corrupt) {
		t.Errorf("corrupt usage log was overwritten:\nbefore: %s\nafter:  %s", corrupt, after)
	}
	assertUsageIncomplete(t, getUsage(t, server.URL))
}

func TestGetUsage_LogHealthyAndRecordingAgain_ClearsIncompleteness(t *testing.T) {
	dataDir := seedDataDir(t)
	logPath := filepath.Join(dataDir, "usage-log.json")
	if err := os.WriteFile(logPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	client := usageReportingClient()
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	saveJobListing(t, server.URL, "Acme Corp")
	assertUsageIncomplete(t, getUsage(t, server.URL))

	// The user removes the broken log; the next recorded call succeeds.
	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	client.usage = []generation.CallUsage{
		{CallType: "ral_estimation", InputTokens: 10, OutputTokens: 5, EstimatedCostUSD: 0.001},
	}
	saveJobListing(t, server.URL, "Globex")

	got := getUsage(t, server.URL)
	assertUsageComplete(t, got)
	if calls, _ := got["calls"].([]any); len(calls) != 1 {
		t.Errorf("expected the 1 call recorded after recovery, got %v", got["calls"])
	}
}

func TestSaveJobListing_UsageLogWriteFails_SaveStillSucceedsAndLogIsUntouched(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permission bits, so the usage log write can't be made to fail")
	}
	dataDir := seedDataDir(t)
	logPath := filepath.Join(dataDir, "usage-log.json")
	if err := os.WriteFile(logPath, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"jobs", "applications"} {
		if err := os.MkdirAll(filepath.Join(dataDir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Usage-log writes are atomic (temp file + rename in the data dir), so
	// only an unwritable data dir makes them fail; the Job Listing and its
	// Application still save into their own writable subdirectories.
	if err := os.Chmod(dataDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dataDir, 0o755) })
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, usageReportingClient()))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Acme Corp",
		"jobDescription": "Some role.",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected the save to still succeed with 201, got %d", resp.StatusCode)
	}
	if got, _ := os.ReadFile(logPath); string(got) != "[]" {
		t.Errorf("expected the usage log left untouched by the failed write, got %q", got)
	}
}
