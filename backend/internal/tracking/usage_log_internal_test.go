package tracking

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

type drainingUsage struct{ calls []generation.CallUsage }

func (d *drainingUsage) DrainUsage() []generation.CallUsage {
	calls := d.calls
	d.calls = nil
	return calls
}

// A failed usage-log write is recorded on disk, so the running total stays
// marked incomplete across a restart, and a later healthy write clears it
// (issue #102).
func TestRecordStandaloneUsage_FailedLogWriteMarksTotalIncompleteUntilALaterWriteSucceeds(t *testing.T) {
	dataDir := t.TempDir()
	logPath := filepath.Join(dataDir, usageLogFile)
	if err := os.WriteFile(logPath, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeUsageLogFile = func(string, []byte, os.FileMode) error { return errors.New("disk full") }
	t.Cleanup(func() { writeUsageLogFile = originalWriteUsageLogFile })

	RecordStandaloneUsage(dataDir, &drainingUsage{calls: []generation.CallUsage{{CallType: "ral_estimation", InputTokens: 10}}})

	if got, _ := os.ReadFile(logPath); string(got) != "[]" {
		t.Errorf("expected the usage log left untouched by the failed write, got %q", got)
	}
	total, err := TotalUsage(dataDir)
	if err != nil {
		t.Fatalf("TotalUsage: %v", err)
	}
	if total.IncompleteReason == "" {
		t.Fatal("expected the total marked incomplete after a failed log write, got a complete total")
	}

	writeUsageLogFile = originalWriteUsageLogFile
	RecordStandaloneUsage(dataDir, &drainingUsage{calls: []generation.CallUsage{{CallType: "ral_estimation", InputTokens: 20}}})

	total, err = TotalUsage(dataDir)
	if err != nil {
		t.Fatalf("TotalUsage: %v", err)
	}
	if total.IncompleteReason != "" {
		t.Errorf("expected a healthy write to clear the incompleteness, still got %q", total.IncompleteReason)
	}
}

var originalWriteUsageLogFile = writeUsageLogFile
