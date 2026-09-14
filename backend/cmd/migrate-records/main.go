// Command migrate-records brings every Job Listing and Application record
// under a data directory to the current schema version (issue #100,
// ADR-0034). It is a thin wrapper over tracking.MigrateRecords: flag
// parsing and printing the report, nothing else.
//
// By default it is a dry run that only reports. Pass -write to apply.
//
// Exit status: 0 when nothing is pending (or -write applied everything), 1
// on an error (an unparseable record, a record at a newer schema version,
// an I/O failure — nothing is written in the first two cases), 2 on a bad
// flag, and 3 when a dry run found records still to migrate.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

const exitPending = 3

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("migrate-records", flag.ContinueOnError)
	flags.SetOutput(stderr)
	defaultDataDir := os.Getenv("DATA_DIR")
	if defaultDataDir == "" {
		defaultDataDir = "data"
	}
	dataDir := flags.String("data-dir", defaultDataDir, "data directory holding jobs/ and applications/ (default $DATA_DIR, else ./data)")
	write := flags.Bool("write", false, "apply the migration; without it, only report what would change")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		absDataDir = *dataDir
	}
	if *write {
		fmt.Fprintf(stdout, "Migrating records under %s to schema version %d.\n", absDataDir, tracking.CurrentSchemaVersion)
	} else {
		fmt.Fprintf(stdout, "Dry run over %s: nothing will be written (pass -write to apply).\n", absDataDir)
	}
	if _, statErr := os.Stat(absDataDir); os.IsNotExist(statErr) {
		fmt.Fprintf(stdout, "Note: %s does not exist.\n", absDataDir)
	}

	report, err := tracking.MigrateRecords(*dataDir, tracking.MigrationOptions{DryRun: !*write})
	if err != nil {
		fmt.Fprintf(stderr, "migrate-records: %v\n", err)
		// Every record is read and checked before the first write, so only
		// an I/O failure while writing can leave the run half-applied — and
		// each record it did write was replaced atomically.
		if *write {
			fmt.Fprintln(stderr, "If this failed while writing, some records may already be migrated; run the dry run again to see what is still pending.")
		} else {
			fmt.Fprintln(stderr, "Nothing was written.")
		}
		return 1
	}
	printReport(stdout, report)

	switch {
	case !report.Pending():
		fmt.Fprintf(stdout, "\nAll %d records are already at schema version %d. Nothing to do.\n", report.Scanned, tracking.CurrentSchemaVersion)
		return 0
	case report.DryRun:
		fmt.Fprintf(stdout, "\n%d of %d records would be migrated. Re-run with -write to apply.\n", len(report.Migrated), report.Scanned)
		return exitPending
	default:
		fmt.Fprintf(stdout, "\nMigrated %d of %d records.\n", len(report.Migrated), report.Scanned)
		return 0
	}
}

func printReport(w io.Writer, report tracking.MigrationReport) {
	fmt.Fprintf(w, "Scanned %d records.\n", report.Scanned)
	for _, m := range report.Migrated {
		fmt.Fprintf(w, "\n%s (%s): schema version %d -> %d\n", m.Path, m.Kind, m.FromVersion, m.ToVersion)
		for _, f := range m.Backfilled {
			fmt.Fprintf(w, "  backfilled   %s = %s\n", f.Field, f.Value)
		}
		for _, f := range m.Unknowable {
			fmt.Fprintf(w, "  left empty   %s: %s\n", f.Field, f.Reason)
		}
		if len(m.Backfilled) == 0 && len(m.Unknowable) == 0 {
			fmt.Fprintln(w, "  version stamp only")
		}
	}
	if len(report.Inconsistencies) > 0 {
		fmt.Fprintln(w, "\nInconsistencies (reported, not changed):")
		for _, inc := range report.Inconsistencies {
			fmt.Fprintf(w, "  %s: %s\n", inc.Path, inc.Problem)
		}
	}
}
