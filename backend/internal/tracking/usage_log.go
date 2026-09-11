package tracking

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/atomicfile"
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
		markUsageIncomplete(dataDir, fmt.Sprintf("%d Claude API call(s) were not recorded because the usage log (%s) is unreadable: %v", len(calls), usageLogFile, err))
		return
	}
	existing = append(existing, calls...)

	data, err := json.MarshalIndent(existing, "", "  ")
	if err == nil {
		err = atomicfile.WriteFile(path, data, 0o644)
	}
	if err != nil {
		log.Printf("usage log: failed to record %d Claude API call(s) to %s: %v", len(calls), path, err)
		markUsageIncomplete(dataDir, fmt.Sprintf("%d Claude API call(s) could not be written to the usage log (%s): %v", len(calls), usageLogFile, err))
		return
	}

	// The log was read and written cleanly, so it's healthy again: clear
	// any earlier marker rather than branding the total suspect forever.
	if err := os.Remove(filepath.Join(dataDir, usageIncompleteFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("usage log: failed to clear incompleteness marker: %v", err)
	}
}

// usageIncompleteFile is a sidecar beside the usage log recording that
// some usage has been lost (issue #102). It lives outside the log because
// the log itself is exactly what's unwritable or unparseable when it's
// needed, and it's on disk (not in memory) so a loss in one session is
// still disclosed after a restart.
const usageIncompleteFile = "usage-log.incomplete.json"

type usageIncompleteMarker struct {
	Reason     string    `json:"reason"`
	RecordedAt time.Time `json:"recordedAt"`
}

// markUsageIncomplete best-effort persists the incompleteness marker. If
// that write fails too, the server log line already emitted by the caller
// is the last line of defence.
func markUsageIncomplete(dataDir, reason string) {
	path := filepath.Join(dataDir, usageIncompleteFile)
	data, err := json.MarshalIndent(usageIncompleteMarker{Reason: reason, RecordedAt: time.Now().UTC()}, "", "  ")
	if err == nil {
		err = atomicfile.WriteFile(path, data, 0o644)
	}
	if err != nil {
		log.Printf("usage log: failed to write incompleteness marker %s: %v", path, err)
	}
}

// readUsageIncompleteReason returns the persisted marker's reason, or ""
// when no marker exists. A marker that exists but can't be read still
// counts as incomplete.
func readUsageIncompleteReason(dataDir string) string {
	data, err := os.ReadFile(filepath.Join(dataDir, usageIncompleteFile))
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	var marker usageIncompleteMarker
	if err != nil || json.Unmarshal(data, &marker) != nil || marker.Reason == "" {
		return fmt.Sprintf("Some Claude API usage could not be recorded (see %s and the backend log).", usageIncompleteFile)
	}
	return marker.Reason
}

// TotalUsage aggregates every Claude API call this dataDir has ever
// recorded usage for — every persisted GenerationRecord's Calls breakdown
// (Generations across every Application) plus the standalone usage log
// (RAL/Method/Contact calls made outside a Generation) — into one running
// total (PRD story 11).
func TotalUsage(dataDir string) (UsageTotal, error) {
	listings, err := List(dataDir)
	if err != nil {
		return UsageTotal{}, err
	}

	var calls []generation.CallUsage
	for _, lw := range listings {
		for _, g := range lw.Application.Generations {
			calls = append(calls, g.Usage.Calls...)
		}
	}
	total := UsageTotal{}
	logged, err := ReadUsageLog(dataDir)
	if err != nil {
		total.IncompleteReason = fmt.Sprintf("The standalone usage log (%s) is unreadable, so its calls are missing from this total and new ones are not being recorded until it is fixed or removed: %v", usageLogFile, err)
	} else {
		total.IncompleteReason = readUsageIncompleteReason(dataDir)
	}
	calls = append(calls, logged...)

	total.Usage = generation.NewGenerationUsage(calls)
	return total, nil
}

// UsageTotal is TotalUsage's result: the aggregated usage plus, when some
// usage is known to be missing from it, why (issue #102). An empty
// IncompleteReason means the total is complete.
type UsageTotal struct {
	Usage            generation.GenerationUsage
	IncompleteReason string
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
