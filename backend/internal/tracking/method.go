package tracking

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// Client is the seam tracking needs an LLM for: generation.Client's RAL
// Range lookup (a Job Listing's RAL Range, reused from Generation) plus
// Application Method inference from a Job Description (story 5). The real
// implementation is claude.Client, which already satisfies
// generation.Client.
type Client interface {
	generation.Client
	InferApplicationMethod(ctx context.Context, jobDescription string) (ApplicationMethod, error)
}

// knownMethodKinds validates a user correction (story 6) without
// restricting which Kind it moves to or from — unlike Status, there's no
// meaningful progression to enforce here.
var knownMethodKinds = map[ApplicationMethodKind]bool{
	MethodPortal:    true,
	MethodEmail:     true,
	MethodEasyApply: true,
	MethodOther:     true,
}

// UpdateApplicationMethod validates and applies a user correction to the
// Application identified by id (story 6), writing it back to disk.
func UpdateApplicationMethod(dataDir, id string, method ApplicationMethod) (Application, error) {
	if !knownMethodKinds[method.Kind] {
		return Application{}, fmt.Errorf("%w: unknown application method kind %q", ErrValidation, method.Kind)
	}

	application, err := getApplication(dataDir, id)
	if err != nil {
		return Application{}, err
	}
	application.Method = method

	if err := os.WriteFile(filepath.Join(dataDir, applicationsDir, id+".md"), renderApplication(application), 0o644); err != nil {
		return Application{}, err
	}
	return application, nil
}
