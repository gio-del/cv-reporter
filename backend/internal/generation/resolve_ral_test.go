package generation

import (
	"context"
	"testing"
)

// fakeClient is the generation.Client fake for ResolveRAL's own test seam,
// mirroring api_test's fakeGenerationClient pattern but scoped to this
// package's narrower Client interface.
type fakeClient struct {
	estimateRAL func(ctx context.Context, jobDescription string) (RALRange, error)
}

func (f *fakeClient) SelectAndRewrite(ctx context.Context, req SelectionRequest) (SelectionResult, error) {
	return SelectionResult{}, nil
}

func (f *fakeClient) SelectOnly(ctx context.Context, req SelectionRequest) (SelectionResult, error) {
	return SelectionResult{}, nil
}

func (f *fakeClient) DraftCoverLetter(ctx context.Context, req CoverLetterRequest) (CoverLetterResult, error) {
	return CoverLetterResult{}, nil
}

func (f *fakeClient) EstimateRAL(ctx context.Context, jobDescription string) (RALRange, error) {
	if f.estimateRAL == nil {
		return RALRange{Source: RALSourceNA}, nil
	}
	return f.estimateRAL(ctx, jobDescription)
}

func TestResolveRAL_OnlyDescriptionStates_ReturnsStatedFromDescription(t *testing.T) {
	client := &fakeClient{}
	ral, err := ResolveRAL(context.Background(), "RAL 45,000 - 55,000 EUR", "", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ral.Source != RALSourceStated {
		t.Errorf("expected source stated, got %v", ral.Source)
	}
	if ral.Min == nil || *ral.Min != 45000 || ral.Max == nil || *ral.Max != 55000 {
		t.Errorf("expected 45000-55000, got min=%v max=%v", ral.Min, ral.Max)
	}
}

func TestResolveRAL_OnlyListingStates_ReturnsStatedFromListing(t *testing.T) {
	client := &fakeClient{}
	ral, err := ResolveRAL(context.Background(), "Go backend engineer, remote friendly.", "63,2K € /yr - 70,8K € /yr", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ral.Source != RALSourceStated {
		t.Errorf("expected source stated, got %v", ral.Source)
	}
	if ral.Min == nil || *ral.Min != 63200 || ral.Max == nil || *ral.Max != 70800 {
		t.Errorf("expected 63200-70800, got min=%v max=%v", ral.Min, ral.Max)
	}
}

func TestResolveRAL_BothStateOverlappingFigures_ReturnsStatedFromDescriptionOnly(t *testing.T) {
	client := &fakeClient{}
	ral, err := ResolveRAL(context.Background(), "RAL 60,000 - 70,000 EUR", "63,2K € /yr - 70,8K € /yr", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ral.Source != RALSourceStated {
		t.Errorf("expected source stated, got %v", ral.Source)
	}
	if ral.Min == nil || *ral.Min != 60000 || ral.Max == nil || *ral.Max != 70000 {
		t.Errorf("expected the description's own 60000-70000, got min=%v max=%v", ral.Min, ral.Max)
	}
	if ral.DescriptionStated != nil || ral.ListingStated != nil {
		t.Errorf("expected no sub-figures outside conflict, got description=%v listing=%v", ral.DescriptionStated, ral.ListingStated)
	}
}

func TestResolveRAL_BothStateNonOverlappingFigures_ReturnsConflictWithBothFigures(t *testing.T) {
	client := &fakeClient{}
	ral, err := ResolveRAL(context.Background(), "RAL Fino a 63.000 EUR", "70,8K € /yr - 80,8K € /yr", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ral.Source != RALSourceConflict {
		t.Fatalf("expected source conflict, got %v", ral.Source)
	}
	if ral.Min != nil || ral.Max != nil {
		t.Errorf("expected no auto-picked min/max on conflict, got min=%v max=%v", ral.Min, ral.Max)
	}
	if ral.DescriptionStated == nil {
		t.Fatal("expected DescriptionStated to be populated")
	}
	if ral.DescriptionStated.Min != 63000 || ral.DescriptionStated.Max != 63000 || ral.DescriptionStated.Currency != "EUR" {
		t.Errorf("unexpected DescriptionStated: %+v", ral.DescriptionStated)
	}
	if ral.ListingStated == nil {
		t.Fatal("expected ListingStated to be populated")
	}
	if ral.ListingStated.Min != 70800 || ral.ListingStated.Max != 80800 || ral.ListingStated.Currency != "EUR" {
		t.Errorf("unexpected ListingStated: %+v", ral.ListingStated)
	}
}

func TestResolveRAL_NeitherStates_FallsBackToEstimateRAL(t *testing.T) {
	called := false
	client := &fakeClient{
		estimateRAL: func(ctx context.Context, jobDescription string) (RALRange, error) {
			called = true
			min, max := 50000, 60000
			return RALRange{Min: &min, Max: &max, Currency: "EUR", Source: RALSourceEstimated}, nil
		},
	}
	ral, err := ResolveRAL(context.Background(), "Go backend engineer, remote friendly.", "", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected EstimateRAL to be called when neither source states a figure")
	}
	if ral.Source != RALSourceEstimated {
		t.Errorf("expected source estimated, got %v", ral.Source)
	}
	if ral.Min == nil || *ral.Min != 50000 || ral.Max == nil || *ral.Max != 60000 {
		t.Errorf("expected the estimate's 50000-60000, got min=%v max=%v", ral.Min, ral.Max)
	}
}
