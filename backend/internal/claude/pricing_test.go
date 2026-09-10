package claude

import (
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
