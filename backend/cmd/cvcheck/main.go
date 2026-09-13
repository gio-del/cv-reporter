// Command cvcheck runs the web app's automated Generation quality checks
// from the command line, so the tailor-cv skill can apply the same scrutiny
// to a skill-run Generation that the app applies to its own (issue #104,
// ADR-0028). It calls the generation package's exported check facades —
// the same code the backend runs — rather than reimplementing any check.
//
// Every check is pure and offline (no Claude call, no network, no running
// backend), which is why this is a CLI the skill shells out to rather than
// an HTTP call to the container.
//
// Usage:
//
//	cvcheck groundedness --selection output/<slug>/selection.json [--data-dir data] [--json]
//
// Exit status: 0 = ran, nothing flagged; 1 = ran, something flagged;
// 2 = the check could not run (bad usage, missing/malformed artifact,
// artifact not traceable to Master Data). None of these is meant to stop
// the skill's pipeline: flags are information for the human checkpoint,
// not failures.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

const (
	exitClean       = 0
	exitFlagged     = 1
	exitUnavailable = 2
)

const usage = `usage:
  cvcheck groundedness --selection <output/<slug>/selection.json> [--data-dir data] [--json]`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return unavailable(stderr, errors.New("no subcommand given\n"+usage))
	}
	switch args[0] {
	case "groundedness":
		return runGroundedness(args[1:], stdout, stderr)
	default:
		return unavailable(stderr, fmt.Errorf("unknown subcommand %q\n%s", args[0], usage))
	}
}

// unavailable reports that a check could not run at all — distinct from a
// check that ran and flagged something, so a tooling or input problem is
// never mistaken for a verdict about the CV.
func unavailable(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "cvcheck: check unavailable: %v\n", err)
	return exitUnavailable
}

func runGroundedness(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("groundedness", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	selectionPath := fs.String("selection", "", "path to the Selection+Rewrite artifact (output/<slug>/selection.json)")
	dataDir := fs.String("data-dir", "data", "path to the Master Data directory the selection was drawn from")
	asJSON := fs.Bool("json", false, "print the GroundednessResult as JSON instead of human-readable text")
	if err := fs.Parse(args); err != nil {
		return unavailable(stderr, fmt.Errorf("%v\n%s", err, usage))
	}
	if *selectionPath == "" {
		return unavailable(stderr, errors.New("--selection is required\n"+usage))
	}

	selection, err := readSelection(*selectionPath)
	if err != nil {
		return unavailable(stderr, err)
	}

	result, err := generation.CheckSelectionGroundedness(*dataDir, selection)
	if err != nil {
		return unavailable(stderr, fmt.Errorf("groundedness: %w", err))
	}

	if *asJSON {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			return unavailable(stderr, fmt.Errorf("encoding result: %w", err))
		}
	} else {
		printGroundedness(stdout, selection, result)
	}

	if len(result.Bullets) > 0 {
		return exitFlagged
	}
	return exitClean
}

// readSelection decodes the Selection+Rewrite artifact strictly into
// generation.SelectionResult: an unknown field (e.g. "rewrittenText" for
// "rewritten") is an error rather than a silently empty rewrite that would
// pass the check without having been checked.
func readSelection(path string) (generation.SelectionResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return generation.SelectionResult{}, fmt.Errorf("reading selection artifact: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(content))
	dec.DisallowUnknownFields()
	var selection generation.SelectionResult
	if err := dec.Decode(&selection); err != nil {
		return generation.SelectionResult{}, fmt.Errorf("parsing selection artifact %s: %w", path, err)
	}
	return selection, nil
}

func printGroundedness(w io.Writer, selection generation.SelectionResult, result generation.GroundednessResult) {
	total := 0
	for _, e := range selection.Entries {
		total += len(e.Bullets)
	}

	if len(result.Bullets) == 0 {
		fmt.Fprintf(w, "Groundedness: checked %s against their source bullets — no flags.\n", pluralBullets(total))
	} else {
		fmt.Fprintf(w, "Groundedness: %d of %s flagged (non-blocking — look hardest at these at Text Review):\n", len(result.Bullets), pluralBullets(total))
		for _, b := range result.Bullets {
			for _, f := range b.Flags {
				fmt.Fprintf(w, "- %s bullet %d [%s: %s]\n  %q\n", b.EntryID, b.SourceIndex, f.Reason, explainReason(f.Reason), f.Sentence)
			}
		}
	}

	fmt.Fprintln(w, describeLanguage(selection.Language))
}

func pluralBullets(n int) string {
	if n == 1 {
		return "1 rewritten bullet"
	}
	return fmt.Sprintf("%d rewritten bullets", n)
}

func explainReason(r generation.GroundednessReason) string {
	switch r {
	case generation.ReasonNumericMismatch:
		return "states a number or date its source bullet doesn't"
	case generation.ReasonNoSourceMatch:
		return "no plausible match in its source bullet"
	default:
		return "flagged"
	}
}

// describeLanguage applies generation.NormalizeLanguage — the app's own
// supported-set-with-fallback rule — to the language recorded in the
// selection artifact, so the skill writes the same resolved code into the
// assembled data's lang field that the app would.
func describeLanguage(detected string) string {
	resolved := generation.NormalizeLanguage(detected)
	switch {
	case strings.TrimSpace(detected) == "":
		return fmt.Sprintf("Target language: %s (none recorded in the selection artifact; using the default)", resolved)
	case strings.ToLower(strings.TrimSpace(detected)) != resolved:
		return fmt.Sprintf("Target language: %s (detected %q is not supported; falling back to the default)", resolved, detected)
	default:
		return "Target language: " + resolved
	}
}
