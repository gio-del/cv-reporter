package generation

// CallUsage is the token/tool usage and estimated cost of one Claude API
// call, as reported by a Client that implements UsageRecorder (the real
// claude.Client; test fakes need not implement it, in which case usage
// capture is simply skipped — see DrainUsage).
type CallUsage struct {
	// CallType identifies which kind of Claude call this was (e.g.
	// "selection_rewrite", "cover_letter", "ral_estimation",
	// "contact_suggestion") so a lifetime total can be broken down by call
	// type (PRD story 11), not just reported as one opaque number.
	CallType         string  `json:"callType"`
	Model            string  `json:"model"`
	InputTokens      int64   `json:"inputTokens"`
	OutputTokens     int64   `json:"outputTokens"`
	CacheReadTokens  int64   `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens int64   `json:"cacheWriteTokens,omitempty"`
	WebSearchUses    int     `json:"webSearchUses,omitempty"`
	EstimatedCostUSD float64 `json:"estimatedCostUsd"`
}

// UsageRecorder is optionally implemented by a Client to expose the usage
// of every Claude API call it has made since the last drain. Generate (and
// the tracking package, for RAL/Contact calls not tied to a Generation)
// type-asserts against this so cost/usage visibility stays purely additive
// (PRD story 10): a Client that doesn't implement it — every test fake —
// is simply skipped.
type UsageRecorder interface {
	// DrainUsage returns every CallUsage recorded since the last call to
	// DrainUsage (or since the Client was created) and clears its internal
	// accumulator. Implementations must be safe to call even when nothing
	// has been recorded yet (returning nil/empty).
	DrainUsage() []CallUsage
}

// GenerationUsage aggregates every CallUsage one Generate run produced into
// totals plus the by-call-type breakdown Calls preserves, attachable to a
// persisted Generation record.
type GenerationUsage struct {
	InputTokens      int64       `json:"inputTokens"`
	OutputTokens     int64       `json:"outputTokens"`
	CacheReadTokens  int64       `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens int64       `json:"cacheWriteTokens,omitempty"`
	WebSearchUses    int         `json:"webSearchUses,omitempty"`
	EstimatedCostUSD float64     `json:"estimatedCostUsd"`
	Calls            []CallUsage `json:"calls,omitempty"`
}

// aggregateUsage sums calls into one GenerationUsage, keeping the
// individual calls as the by-call-type breakdown.
func aggregateUsage(calls []CallUsage) GenerationUsage {
	var g GenerationUsage
	for _, c := range calls {
		g.InputTokens += c.InputTokens
		g.OutputTokens += c.OutputTokens
		g.CacheReadTokens += c.CacheReadTokens
		g.CacheWriteTokens += c.CacheWriteTokens
		g.WebSearchUses += c.WebSearchUses
		g.EstimatedCostUSD += c.EstimatedCostUSD
	}
	g.Calls = calls
	return g
}

// DrainUsage best-effort drains usageSource's accumulated Claude API call
// usage if it implements UsageRecorder. Usage capture must never affect
// the call it's wrapping (PRD story 9/10): a Client that doesn't implement
// UsageRecorder, or a recorder that panics, yields no usage rather than an
// error.
func DrainUsage(usageSource any) (calls []CallUsage) {
	defer func() { recover() }()
	recorder, ok := usageSource.(UsageRecorder)
	if !ok {
		return nil
	}
	return recorder.DrainUsage()
}
