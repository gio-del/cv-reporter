package tracking

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

const usageLogFile = "usage-log.json"

// usageLogMu serializes read-modify-write access to dataDir's usage log
// across concurrent requests (RAL resolution, Contact suggestion, ...).
var usageLogMu sync.Mutex

// RecordStandaloneUsage best-effort appends any Claude API usage client has
// accumulated since its last drain to dataDir's usage log file — for calls
// made outside a Generation (RAL Range / Application Method resolution,
// Contact suggestion, the Selection preview), which have no GenerationRecord
// to persist usage on (PRD "Claude API cost/usage visibility", story 10).
// Never returns an error: usage logging must never affect the call it's
// logging. A failure is logged server-side instead (issue #102).
//
// The append is a read-modify-write, so it only proceeds when the existing
// log was read cleanly or doesn't exist yet: appending onto the empty base
// an unreadable log yields, then writing that, would wipe every call
// accumulated so far (issue #102). The unreadable file is left untouched.
func RecordStandaloneUsage(dataDir string, client any) {
	calls := generation.DrainUsage(client)
	if len(calls) == 0 {
		return
	}

	usageLogMu.Lock()
	defer usageLogMu.Unlock()

	path := filepath.Join(dataDir, usageLogFile)
	existing, err := ReadUsageLog(dataDir)
	if err != nil {
		log.Printf("usage log: not recording %d Claude API call(s), refusing to overwrite unreadable %s: %v", len(calls), path, err)
		return
	}
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
	logged, _ := ReadUsageLog(dataDir)
	calls = append(calls, logged...)

	return generation.NewGenerationUsage(calls), nil
}

// ReadUsageLog returns every standalone (non-Generation) CallUsage recorded
// so far for dataDir. A log that doesn't exist yet is normal (a fresh
// install) and returns nil with no error; a log that exists but can't be
// read or parsed returns an error, so "nothing recorded" and "record
// broken" stay distinguishable (issue #102).
func ReadUsageLog(dataDir string) ([]generation.CallUsage, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, usageLogFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var calls []generation.CallUsage
	if err := json.Unmarshal(data, &calls); err != nil {
		return nil, fmt.Errorf("parse %s: %w", usageLogFile, err)
	}
	return calls, nil
}
