package tracking

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

const usageLogFile = "usage-log.json"

// usageLogMu serializes read-modify-write access to dataDir's usage log
// across concurrent requests (RAL resolution, Contact suggestion, ...).
var usageLogMu sync.Mutex

// recordStandaloneUsage best-effort appends any Claude API usage client has
// accumulated since its last drain to dataDir's usage log file — for calls
// made outside a Generation (RAL Range / Application Method resolution,
// Contact suggestion), which have no GenerationRecord to persist usage on
// (PRD "Claude API cost/usage visibility", story 10). Silently does nothing
// on any failure: usage logging must never affect the call it's logging.
func recordStandaloneUsage(dataDir string, client any) {
	calls := generation.DrainUsage(client)
	if len(calls) == 0 {
		return
	}

	usageLogMu.Lock()
	defer usageLogMu.Unlock()

	path := filepath.Join(dataDir, usageLogFile)
	existing := ReadUsageLog(dataDir)
	existing = append(existing, calls...)

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

// TotalUsage aggregates every Claude API call this dataDir has ever
// recorded usage for — every persisted GenerationRecord's Calls breakdown
// (Generations across every Application) plus the standalone usage log
// (RAL/Method/Contact calls made outside a Generation) — into one running
// total (PRD story 11).
func TotalUsage(dataDir string) (generation.GenerationUsage, error) {
	listings, err := List(dataDir)
	if err != nil {
		return generation.GenerationUsage{}, err
	}

	var calls []generation.CallUsage
	for _, lw := range listings {
		for _, g := range lw.Application.Generations {
			calls = append(calls, g.Usage.Calls...)
		}
	}
	calls = append(calls, ReadUsageLog(dataDir)...)

	return generation.NewGenerationUsage(calls), nil
}

// ReadUsageLog returns every standalone (non-Generation) CallUsage recorded
// so far for dataDir, or nil if the log doesn't exist yet or can't be read
// — usage visibility is best-effort throughout, never a source of errors.
func ReadUsageLog(dataDir string) []generation.CallUsage {
	data, err := os.ReadFile(filepath.Join(dataDir, usageLogFile))
	if err != nil {
		return nil
	}
	var calls []generation.CallUsage
	if err := json.Unmarshal(data, &calls); err != nil {
		return nil
	}
	return calls
}
