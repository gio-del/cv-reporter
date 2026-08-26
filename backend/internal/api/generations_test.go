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
)

// fakeGenerationClient is the generation.Client fake used to test the
// /api/generations seam without calling the real Claude API, per the PRD's
// Testing Decisions.
type fakeGenerationClient struct {
	selectAndRewrite func(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error)
	draftCoverLetter func(ctx context.Context, req generation.CoverLetterRequest) (generation.CoverLetterResult, error)
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
