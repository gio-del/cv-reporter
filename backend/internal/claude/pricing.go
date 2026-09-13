package claude

import (
	"log"
	"strings"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
)

// modelPricing is per-million-token USD rates for one model — the single
// source of truth estimateCost draws from (PRD story 8: update here when
// Anthropic's published pricing changes, not at each call site).
type modelPricing struct {
	inputPerMTok      float64
	outputPerMTok     float64
	cacheReadPerMTok  float64
	cacheWritePerMTok float64
}

// webSearchPerUse is a flat per-tool-invocation rate (distinct from token
// cost), added on top of token pricing for calls that used the web search
// tool (EstimateRAL, SuggestContact) — PRD story 6, Further Notes.
const webSearchPerUse = 10.0 / 1000 // $10 per 1,000 searches

// pricingTable holds every model a Client can be configured to call, keyed
// by alias. Rates are Anthropic's published first-party prices (base
// input, output, cache hits, 5-minute cache writes), hand-maintained.
//
// A model absent from
// this table (e.g. an override naming a model not recorded yet) still
// returns 0 rather than panicking or erroring, but logs a warning the first
// time it is seen, so incomplete cost figures surface in the log.
var pricingTable = map[anthropic.Model]modelPricing{
	anthropic.ModelClaudeSonnet5: {
		inputPerMTok:      2.00,
		outputPerMTok:     10.00,
		cacheReadPerMTok:  0.20,
		cacheWritePerMTok: 2.50,
	},
	anthropic.ModelClaudeHaiku4_5: {
		inputPerMTok:      1.00,
		outputPerMTok:     5.00,
		cacheReadPerMTok:  0.10,
		cacheWritePerMTok: 1.25,
	},
	anthropic.ModelClaudeOpus5: {
		inputPerMTok:      5.00,
		outputPerMTok:     25.00,
		cacheReadPerMTok:  0.50,
		cacheWritePerMTok: 6.25,
	},
}

// lookupPricing resolves model to its pricingTable entry. The API reports
// the model it actually ran, which can be a dated snapshot id
// (claude-haiku-4-5-20251001) rather than the alias requested, so after an
// exact match fails this falls back to the longest table key that model
// extends with a "-<digits>" date suffix. Any other extension (e.g. a
// hypothetical point release "claude-opus-5-1") is a different model and
// is deliberately not matched.
func lookupPricing(model anthropic.Model) (modelPricing, bool) {
	if pricing, ok := pricingTable[model]; ok {
		return pricing, true
	}
	var best anthropic.Model
	for key := range pricingTable {
		suffix, ok := strings.CutPrefix(string(model), string(key)+"-")
		if !ok || !isDateSuffix(suffix) {
			continue
		}
		if len(key) > len(best) {
			best = key
		}
	}
	if best == "" {
		return modelPricing{}, false
	}
	return pricingTable[best], true
}

// isDateSuffix reports whether s looks like a snapshot date (YYYYMMDD).
func isDateSuffix(s string) bool {
	if len(s) != 8 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// warnedUnpriced records every model warnUnpricedOnce has already logged,
// so a long-running server warns once per model rather than on every call.
var warnedUnpriced sync.Map

func warnUnpricedOnce(model anthropic.Model) {
	if _, already := warnedUnpriced.LoadOrStore(model, struct{}{}); already {
		return
	}
	log.Printf("warning: no pricing entry for Claude model %q — its calls are estimated at $0 until internal/claude/pricing.go is updated", model)
}

// estimateCost computes usage's estimated cost (PRD story 7: an estimate,
// never treated as an authoritative bill) from pricingTable, plus
// webSearchUses's flat per-use rate for calls that used the web search
// tool.
func estimateCost(model anthropic.Model, usage anthropic.Usage, webSearchUses int64) float64 {
	pricing, ok := lookupPricing(model)
	if !ok {
		warnUnpricedOnce(model)
		return 0
	}
	cost := float64(usage.InputTokens)/1_000_000*pricing.inputPerMTok +
		float64(usage.OutputTokens)/1_000_000*pricing.outputPerMTok +
		float64(usage.CacheReadInputTokens)/1_000_000*pricing.cacheReadPerMTok +
		float64(usage.CacheCreationInputTokens)/1_000_000*pricing.cacheWritePerMTok
	cost += float64(webSearchUses) * webSearchPerUse
	return cost
}
