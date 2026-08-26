package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
	"github.com/gio-del/cv-reporter/backend/internal/generation"
	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

// fakeGenerationClient is the tracking.Client (which embeds
// generation.Client) fake used to test the /api/generations and
// /api/job-listings seams without calling the real Claude API, per the
// PRD's Testing Decisions.
type fakeGenerationClient struct {
	selectAndRewrite       func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error)
	draftCoverLetter       func(ctx context.Context, req generation.CoverLetterRequest) (generation.CoverLetterResult, error)
	estimateRAL            func(ctx context.Context, jobDescription string) (generation.RALRange, error)
	inferApplicationMethod func(ctx context.Context, jobDescription string) (tracking.ApplicationMethod, error)
	suggestContact         func(ctx context.Context, company, jobDescription string) (tracking.Contact, error)
}

func (f *fakeGenerationClient) SelectAndRewrite(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
	return f.selectAndRewrite(ctx, req)
}

func (f *fakeGenerationClient) DraftCoverLetter(ctx context.Context, req generation.CoverLetterRequest) (generation.CoverLetterResult, error) {
	if f.draftCoverLetter == nil {
		return generation.CoverLetterResult{Body: "Dear Hiring Manager,\n\nThank you for your consideration."}, nil
	}
	return f.draftCoverLetter(ctx, req)
}

func (f *fakeGenerationClient) EstimateRAL(ctx context.Context, jobDescription string) (generation.RALRange, error) {
	if f.estimateRAL == nil {
		return generation.RALRange{Source: generation.RALSourceNA}, nil
	}
	return f.estimateRAL(ctx, jobDescription)
}

func (f *fakeGenerationClient) InferApplicationMethod(ctx context.Context, jobDescription string) (tracking.ApplicationMethod, error) {
	if f.inferApplicationMethod == nil {
		return tracking.ApplicationMethod{Kind: tracking.MethodOther}, nil
	}
	return f.inferApplicationMethod(ctx, jobDescription)
}

func (f *fakeGenerationClient) SuggestContact(ctx context.Context, company, jobDescription string) (tracking.Contact, error) {
	if f.suggestContact == nil {
		return tracking.Contact{}, nil
	}
	return f.suggestContact(ctx, company, jobDescription)
}

func TestCreateGeneration_WithJobDescription_ReturnsTailoredSelection(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			if req.JobDescription != "Looking for a Go backend engineer." {
				t.Errorf("expected job description to reach the client, got %q", req.JobDescription)
			}
			return generation.SelectionResult{
				Entries: []generation.SelectedEntry{
					{
						EntryID: "experience/quantyca-amplifon",
						Reason:  "Directly relevant AI platform experience.",
						Bullets: []generation.SelectedBullet{
							{
								SourceIndex: 0,
								Source:      "Designed and built an AI Platform.",
								Rewritten:   "Designed and built an AI platform for backend engineering use cases.",
							},
						},
					},
				},
			}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	payload := map[string]any{"jobDescription": "Looking for a Go backend engineer."}
	resp := postJSON(t, server.URL+"/api/generations", payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "tailored" {
		t.Errorf("expected mode tailored, got %v", result["mode"])
	}
	selection, ok := result["selection"].(map[string]any)
	if !ok {
		t.Fatalf("expected a selection object, got %v", result["selection"])
	}
	entries, ok := selection["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("expected 1 selected entry, got %v", selection["entries"])
	}
	entry := entries[0].(map[string]any)
	if entry["entryId"] != "experience/quantyca-amplifon" {
		t.Errorf("expected entryId experience/quantyca-amplifon, got %v", entry["entryId"])
	}
}

func TestCreateGeneration_NoJobDescription_ReturnsDefaultModeWithoutCallingClient(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			t.Fatal("expected Default Mode not to call the Client for Selection")
			return generation.SelectionResult{}, nil
		},
		draftCoverLetter: func(ctx context.Context, req generation.CoverLetterRequest) (generation.CoverLetterResult, error) {
			t.Fatal("expected Default Mode not to call the Client for a Cover Letter")
			return generation.CoverLetterResult{}, nil
		},
		estimateRAL: func(ctx context.Context, jobDescription string) (generation.RALRange, error) {
			t.Fatal("expected Default Mode not to call the Client for a RAL Range")
			return generation.RALRange{}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/generations", map[string]any{})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["mode"] != "default" {
		t.Errorf("expected mode default, got %v", result["mode"])
	}
	if _, ok := result["coverLetter"]; ok {
		t.Errorf("expected no coverLetter in Default Mode, got %v", result["coverLetter"])
	}
	if _, ok := result["ral"]; ok {
		t.Errorf("expected no ral in Default Mode, got %v", result["ral"])
	}
	selection := result["selection"].(map[string]any)
	entries, ok := selection["entries"].([]any)
	if !ok || len(entries) != 2 {
		t.Fatalf("expected all 2 seeded entries in Default Mode, got %v", selection["entries"])
	}
	for _, e := range entries {
		entry := e.(map[string]any)
		for _, b := range entry["bullets"].([]any) {
			bullet := b.(map[string]any)
			if bullet["source"] != bullet["rewritten"] {
				t.Errorf("expected Default Mode to skip Rewrite, got source=%v rewritten=%v", bullet["source"], bullet["rewritten"])
			}
		}
	}
}

func TestCreateGeneration_ClientInventsUnknownEntry_Returns502(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			return generation.SelectionResult{
				Entries: []generation.SelectedEntry{
					{EntryID: "experience/does-not-exist", Bullets: []generation.SelectedBullet{{SourceIndex: 0, Source: "made up", Rewritten: "made up"}}},
				},
			}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/generations", map[string]any{"jobDescription": "Anything"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
}

func TestCreateGeneration_ClientAltersSourceBulletText_Returns502(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			return generation.SelectionResult{
				Entries: []generation.SelectedEntry{
					{
						EntryID: "experience/quantyca-amplifon",
						Bullets: []generation.SelectedBullet{
							{SourceIndex: 0, Source: "This is not the real source bullet.", Rewritten: "Whatever"},
						},
					},
				},
			}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/generations", map[string]any{"jobDescription": "Anything"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502 when the Client's reported source bullet doesn't match Master Data, got %d", resp.StatusCode)
	}
}

func TestCreateGeneration_JobDescriptionURL_FetchesAndPassesTextToClient(t *testing.T) {
	dataDir := seedDataDir(t)
	jdServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><body><h1>Backend Engineer</h1><p>Go experience required.</p></body></html>")
	}))
	defer jdServer.Close()

	var gotJD string
	client := &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			gotJD = req.JobDescription
			return generation.SelectionResult{}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/generations", map[string]any{"jobDescriptionUrl": jdServer.URL})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(gotJD, "Backend Engineer") || !strings.Contains(gotJD, "Go experience required.") {
		t.Errorf("expected fetched page text to reach the client, got %q", gotJD)
	}
	if strings.Contains(gotJD, "<html>") || strings.Contains(gotJD, "<p>") {
		t.Errorf("expected HTML tags to be stripped, got %q", gotJD)
	}
}

func TestCreateGeneration_WithJobDescription_DraftsCoverLetterFromSnippets(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	client := &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			return generation.SelectionResult{}, nil
		},
		draftCoverLetter: func(ctx context.Context, req generation.CoverLetterRequest) (generation.CoverLetterResult, error) {
			if req.JobDescription != "Looking for a Go backend engineer." {
				t.Errorf("expected job description to reach the cover letter client, got %q", req.JobDescription)
			}
			if len(req.Snippets) != 2 {
				t.Errorf("expected the 2 seeded snippets to reach the client, got %v", req.Snippets)
			}
			return generation.CoverLetterResult{
				Body:             "I'm excited to apply... Thank you for your consideration.",
				SourceSnippetIDs: []string{"opening-ai-platforms", "closing-standard"},
			}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/generations", map[string]any{"jobDescription": "Looking for a Go backend engineer."})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	coverLetter, ok := result["coverLetter"].(map[string]any)
	if !ok {
		t.Fatalf("expected a coverLetter object, got %v", result["coverLetter"])
	}
	if coverLetter["body"] != "I'm excited to apply... Thank you for your consideration." {
		t.Errorf("expected cover letter body to round-trip, got %v", coverLetter["body"])
	}
	sourceIDs, ok := coverLetter["sourceSnippetIds"].([]any)
	if !ok || len(sourceIDs) != 2 {
		t.Fatalf("expected 2 source snippet ids, got %v", coverLetter["sourceSnippetIds"])
	}
}

func TestCreateGeneration_CoverLetterReferencesUnknownSnippet_Returns502(t *testing.T) {
	dataDir := seedSnippetsDataDir(t)
	client := &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			return generation.SelectionResult{}, nil
		},
		draftCoverLetter: func(ctx context.Context, req generation.CoverLetterRequest) (generation.CoverLetterResult, error) {
			return generation.CoverLetterResult{
				Body:             "Some prose.",
				SourceSnippetIDs: []string{"does-not-exist"},
			}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/generations", map[string]any{"jobDescription": "Anything"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502 when the Client cites an unknown snippet id, got %d", resp.StatusCode)
	}
}

func TestCreateGeneration_JobDescriptionStatesRAL_SkipsClientAndReportsStated(t *testing.T) {
	dataDir := seedDataDir(t)
	client := &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			return generation.SelectionResult{}, nil
		},
		estimateRAL: func(ctx context.Context, jobDescription string) (generation.RALRange, error) {
			t.Fatal("expected a stated RAL not to call the Client")
			return generation.RALRange{}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	jd := "Go backend engineer. RAL: €45,000 - €55,000."
	resp := postJSON(t, server.URL+"/api/generations", map[string]any{"jobDescription": jd})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	ral, ok := result["ral"].(map[string]any)
	if !ok {
		t.Fatalf("expected a ral object, got %v", result["ral"])
	}
	if ral["source"] != "stated" {
		t.Errorf("expected source stated, got %v", ral["source"])
	}
	if ral["min"] != float64(45000) || ral["max"] != float64(55000) {
		t.Errorf("expected min/max 45000/55000, got %v/%v", ral["min"], ral["max"])
	}
}

func TestCreateGeneration_JobDescriptionOmitsRAL_AsksClientAndReportsEstimated(t *testing.T) {
	dataDir := seedDataDir(t)
	called := false
	client := &fakeGenerationClient{
		selectAndRewrite: func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
			return generation.SelectionResult{}, nil
		},
		estimateRAL: func(ctx context.Context, jobDescription string) (generation.RALRange, error) {
			called = true
			min, max := 50000, 60000
			return generation.RALRange{Min: &min, Max: &max, Currency: "EUR", Source: generation.RALSourceEstimated}, nil
		},
	}
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, client))
	defer server.Close()

	resp := postJSON(t, server.URL+"/api/generations", map[string]any{"jobDescription": "Go backend engineer in Milan."})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !called {
		t.Fatal("expected the Client to be asked to estimate a RAL range")
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	ral := result["ral"].(map[string]any)
	if ral["source"] != "estimated" {
		t.Errorf("expected source estimated, got %v", ral["source"])
	}
}
