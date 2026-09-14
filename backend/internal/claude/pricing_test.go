package claude

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestEstimateCost_KnownModel_TokensOnly(t *testing.T) {
	usage := anthropic.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}

	got := estimateCost(anthropic.ModelClaudeSonnet5, usage, 0)

	want := pricingTable[anthropic.ModelClaudeSonnet5].inputPerMTok + pricingTable[anthropic.ModelClaudeSonnet5].outputPerMTok
	if got != want {
		t.Errorf("estimateCost() = %v, want %v", got, want)
	}
}

func TestEstimateCost_IncludesCacheTokens(t *testing.T) {
	usage := anthropic.Usage{CacheReadInputTokens: 1_000_000, CacheCreationInputTokens: 1_000_000}

	got := estimateCost(anthropic.ModelClaudeSonnet5, usage, 0)

	p := pricingTable[anthropic.ModelClaudeSonnet5]
	want := p.cacheReadPerMTok + p.cacheWritePerMTok
	if got != want {
		t.Errorf("estimateCost() = %v, want %v", got, want)
	}
}

func TestEstimateCost_AddsFlatWebSearchRate(t *testing.T) {
	got := estimateCost(anthropic.ModelClaudeSonnet5, anthropic.Usage{}, 3)

	want := 3 * webSearchPerUse
	if got != want {
		t.Errorf("estimateCost() = %v, want %v", got, want)
	}
}

func TestEstimateCost_UnknownModel_ReturnsZeroNotPanic(t *testing.T) {
	got := estimateCost(anthropic.Model("some-future-model"), anthropic.Usage{InputTokens: 1_000_000}, 5)
	if got != 0 {
		t.Errorf("estimateCost() for unknown model = %v, want 0", got)
	}
}

// oneMTokEach is one million tokens in every billed token category, so
// estimateCost's result is the sum of the four per-MTok rates.
var oneMTokEach = anthropic.Usage{
	InputTokens:              1_000_000,
	OutputTokens:             1_000_000,
	CacheReadInputTokens:     1_000_000,
	CacheCreationInputTokens: 1_000_000,
}

// The rates below are Anthropic's published per-MTok prices (input, output,
// cache hit, 5-minute cache write), written out literally rather than read
// back from pricingTable, so a typo in the table fails here.
func TestEstimateCost_PublishedRates(t *testing.T) {
	tests := []struct {
		model anthropic.Model
		want  float64
	}{
		{anthropic.ModelClaudeSonnet5, 2.00 + 10.00 + 0.20 + 2.50},
		{anthropic.ModelClaudeHaiku4_5, 1.00 + 5.00 + 0.10 + 1.25},
		{anthropic.ModelClaudeOpus5, 5.00 + 25.00 + 0.50 + 6.25},
	}
	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			got := estimateCost(tt.model, oneMTokEach, 0)
			if !approxEqual(got, tt.want) {
				t.Errorf("estimateCost(%s) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestEstimateCost_DatedSnapshotID_UsesAliasPricing(t *testing.T) {
	got := estimateCost(anthropic.ModelClaudeHaiku4_5_20251001, oneMTokEach, 0)

	want := estimateCost(anthropic.ModelClaudeHaiku4_5, oneMTokEach, 0)
	if want == 0 || got != want {
		t.Errorf("estimateCost(%s) = %v, want alias pricing %v", anthropic.ModelClaudeHaiku4_5_20251001, got, want)
	}
}

// A longer id that merely starts with a priced alias but is not a dated
// snapshot of it (a hypothetical point release) must not silently borrow
// that alias's rates — it is a different model and should surface as
// unpriced instead.
func TestEstimateCost_NonDateSuffix_IsNotPrefixMatched(t *testing.T) {
	got := estimateCost(anthropic.Model("claude-opus-5-1"), oneMTokEach, 0)
	if got != 0 {
		t.Errorf("estimateCost(claude-opus-5-1) = %v, want 0 (unpriced)", got)
	}
}

func TestEstimateCost_UnpricedModel_WarnsOnceNamingModel(t *testing.T) {
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() { log.SetOutput(prevOut); log.SetFlags(prevFlags) })

	model := anthropic.Model("unpriced-model-for-warning-test")
	estimateCost(model, oneMTokEach, 0)
	estimateCost(model, oneMTokEach, 0)

	out := buf.String()
	if !strings.Contains(out, string(model)) {
		t.Fatalf("log output = %q, want a warning naming %q", out, model)
	}
	if n := strings.Count(out, string(model)); n != 1 {
		t.Errorf("warning logged %d times, want exactly once", n)
	}
}

func TestEstimateCost_PricedModel_DoesNotWarn(t *testing.T) {
	var buf bytes.Buffer
	prevOut := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prevOut) })

	estimateCost(anthropic.ModelClaudeHaiku4_5_20251001, oneMTokEach, 0)

	if buf.Len() != 0 {
		t.Errorf("log output = %q, want nothing for a priced model", buf.String())
	}
}

func approxEqual(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}
