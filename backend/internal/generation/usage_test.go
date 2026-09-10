package generation

import "testing"

func TestAggregateUsage_SumsAcrossCalls(t *testing.T) {
	calls := []CallUsage{
		{CallType: "selection_rewrite", InputTokens: 1000, OutputTokens: 200, EstimatedCostUSD: 0.01},
		{CallType: "cover_letter", InputTokens: 500, OutputTokens: 100, CacheReadTokens: 50, WebSearchUses: 1, EstimatedCostUSD: 0.02},
	}

	got := aggregateUsage(calls)

	if got.InputTokens != 1500 {
		t.Errorf("InputTokens = %d, want 1500", got.InputTokens)
	}
	if got.OutputTokens != 300 {
		t.Errorf("OutputTokens = %d, want 300", got.OutputTokens)
	}
	if got.CacheReadTokens != 50 {
		t.Errorf("CacheReadTokens = %d, want 50", got.CacheReadTokens)
	}
	if got.WebSearchUses != 1 {
		t.Errorf("WebSearchUses = %d, want 1", got.WebSearchUses)
	}
	if got.EstimatedCostUSD != 0.03 {
		t.Errorf("EstimatedCostUSD = %v, want 0.03", got.EstimatedCostUSD)
	}
	if len(got.Calls) != 2 {
		t.Errorf("Calls = %d entries, want 2 (the breakdown)", len(got.Calls))
	}
}

func TestAggregateUsage_Empty_ReturnsZeroValue(t *testing.T) {
	got := aggregateUsage(nil)
	if got.InputTokens != 0 || got.EstimatedCostUSD != 0 || len(got.Calls) != 0 {
		t.Errorf("expected zero value for no calls, got %+v", got)
	}
}

type fakeUsageRecorder struct {
	calls []CallUsage
}

func (f *fakeUsageRecorder) DrainUsage() []CallUsage {
	drained := f.calls
	f.calls = nil
	return drained
}

func TestDrainUsage_RecorderImplemented_ReturnsAndClears(t *testing.T) {
	rec := &fakeUsageRecorder{calls: []CallUsage{{CallType: "ral_estimation", InputTokens: 10}}}

	got := DrainUsage(rec)
	if len(got) != 1 || got[0].CallType != "ral_estimation" {
		t.Errorf("DrainUsage() = %+v, want the one recorded call", got)
	}

	again := DrainUsage(rec)
	if len(again) != 0 {
		t.Errorf("second drain should be empty (accumulator cleared), got %+v", again)
	}
}

// fakeClientNoRecorder is a stand-in for a test fake that doesn't implement
// UsageRecorder — exactly the shape most existing generation.Client fakes
// in this codebase already take.
type fakeClientNoRecorder struct{}

func TestDrainUsage_ClientDoesNotImplementRecorder_ReturnsNilNotError(t *testing.T) {
	got := DrainUsage(&fakeClientNoRecorder{})
	if got != nil {
		t.Errorf("expected nil for a non-recorder Client, got %+v", got)
	}
}
