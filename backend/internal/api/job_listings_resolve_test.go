package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
	"github.com/gio-del/cv-reporter/backend/internal/generation"
	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

func TestResolveJobListing_RALUnresolved_RetrySucceeds_UpdatesRALLeavesMethodUntouched(t *testing.T) {
	dataDir := seedDataDir(t)

	failingClient := &fakeGenerationClient{
		estimateRAL: func(ctx context.Context, jobDescription string) (generation.RALRange, error) {
			return generation.RALRange{}, errors.New("claude api unreachable")
		},
	}
	setupServer := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, failingClient))
	resp := postJSON(t, setupServer.URL+"/api/job-listings", map[string]any{
		"company":        "Acme Corp",
		"jobDescription": "Go backend engineer, remote.",
	})
	var setup map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&setup); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	setupServer.Close()
	id := setup["jobListing"].(map[string]any)["id"].(string)

	methodInferCalled := false
	min, max := 45000, 55000
	succeedingClient := &fakeGenerationClient{
		estimateRAL: func(ctx context.Context, jobDescription string) (generation.RALRange, error) {
			return generation.RALRange{Min: &min, Max: &max, Currency: "EUR", Source: generation.RALSourceEstimated}, nil
		},
		inferApplicationMethod: func(ctx context.Context, jobDescription string) (tracking.ApplicationMethod, error) {
			methodInferCalled = true
			return tracking.ApplicationMethod{Kind: tracking.MethodOther}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, succeedingClient))
	defer server.Close()

	resolveResp := postJSON(t, server.URL+"/api/job-listings/"+id+"/resolve", nil)
	defer resolveResp.Body.Close()

	if resolveResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resolveResp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resolveResp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	ral := listing["ral"].(map[string]any)
	if ral["source"] != "estimated" {
		t.Errorf("expected RAL now resolved to estimated, got %v", ral["source"])
	}
	if methodInferCalled {
		t.Error("expected Application Method inference NOT to be re-attempted since it was already resolved")
	}
}

func TestResolveJobListing_BothAlreadyResolved_IsNoOp(t *testing.T) {
	dataDir := seedDataDir(t)
	estimateCalled := false
	inferCalled := false
	client := &fakeGenerationClient{
		estimateRAL: func(ctx context.Context, jobDescription string) (generation.RALRange, error) {
			estimateCalled = true
			return generation.RALRange{Source: generation.RALSourceNA}, nil
		},
		inferApplicationMethod: func(ctx context.Context, jobDescription string) (tracking.ApplicationMethod, error) {
			inferCalled = true
			return tracking.ApplicationMethod{Kind: tracking.MethodOther}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	id := saveJobListing(t, server.URL, "Acme Corp")
	estimateCalled = false
	inferCalled = false

	resolveResp := postJSON(t, server.URL+"/api/job-listings/"+id+"/resolve", nil)
	defer resolveResp.Body.Close()

	if resolveResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resolveResp.StatusCode)
	}
	if estimateCalled || inferCalled {
		t.Error("expected resolve to be a no-op (no client calls) when both fields are already resolved")
	}

	var result map[string]any
	if err := json.NewDecoder(resolveResp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	if listing["id"] != id {
		t.Errorf("expected the unchanged listing, got %v", listing)
	}
}

func TestResolveJobListing_UnknownID_Returns404(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/job-listings/does-not-exist/resolve", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestResolveJobListing_FailsAgain_StaysUnresolvedWithout500(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{
		estimateRAL: func(ctx context.Context, jobDescription string) (generation.RALRange, error) {
			return generation.RALRange{}, errors.New("claude api unreachable")
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/job-listings", map[string]any{
		"company":        "Acme Corp",
		"jobDescription": "Go backend engineer, remote.",
	})
	var setup map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&setup); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	id := setup["jobListing"].(map[string]any)["id"].(string)

	resolveResp := postJSON(t, server.URL+"/api/job-listings/"+id+"/resolve", nil)
	defer resolveResp.Body.Close()

	if resolveResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (not an error) when resolution fails again, got %d", resolveResp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resolveResp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	listing := result["jobListing"].(map[string]any)
	ral := listing["ral"].(map[string]any)
	if ral["source"] != "unresolved" {
		t.Errorf("expected RAL to stay unresolved, got %v", ral["source"])
	}
}
