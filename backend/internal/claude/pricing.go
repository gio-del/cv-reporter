package claude

import "github.com/anthropics/anthropic-sdk-go"

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

// pricingTable holds every model this Client is known to call. A model
// absent from this table (e.g. a future model used before pricing is
// recorded here) estimates to zero cost rather than panicking or erroring
// — usage/token counts are still captured and shown, just the cost
// estimate for that call is 0 until the table is updated.
var pricingTable = map[anthropic.Model]modelPricing{
	anthropic.ModelClaudeSonnet5: {
		inputPerMTok:      3.00,
		outputPerMTok:     15.00,
		cacheReadPerMTok:  0.30,
		cacheWritePerMTok: 3.75,
	},
}

// estimateCost computes usage's estimated cost (PRD story 7: an estimate,
// never treated as an authoritative bill) from pricingTable, plus
// webSearchUses's flat per-use rate for calls that used the web search
// tool.
func estimateCost(model anthropic.Model, usage anthropic.Usage, webSearchUses int64) float64 {
	pricing, ok := pricingTable[model]
	if !ok {
		return 0
	}
	cost := float64(usage.InputTokens)/1_000_000*pricing.inputPerMTok +
		float64(usage.OutputTokens)/1_000_000*pricing.outputPerMTok +
		float64(usage.CacheReadInputTokens)/1_000_000*pricing.cacheReadPerMTok +
		float64(usage.CacheCreationInputTokens)/1_000_000*pricing.cacheWritePerMTok
	cost += float64(webSearchUses) * webSearchPerUse
	return cost
}
