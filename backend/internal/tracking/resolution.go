package tracking

import (
	"context"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// resolveRALBestEffort attempts RAL Range resolution, falling back to
// RALSourceUnresolved instead of erroring (PRD "Resilient Job Listing
// Save"): a failure here (Claude API unreachable, no credentials, rate
// limited) must never block persisting an otherwise-valid Job Listing.
// Shared by Save and Resolve so both retry the same way.
func resolveRALBestEffort(ctx context.Context, jobDescription string, client Client) generation.RALRange {
	ral, err := generation.ResolveRAL(ctx, jobDescription, client)
	if err != nil {
		return generation.RALRange{Source: generation.RALSourceUnresolved}
	}
	return ral
}

// resolveApplicationMethodBestEffort attempts Application Method inference,
// falling back to MethodUnresolved instead of erroring, for the same reason
// resolveRALBestEffort does. Shared by Save and Resolve.
func resolveApplicationMethodBestEffort(ctx context.Context, jobDescription string, client Client) ApplicationMethod {
	method, err := client.InferApplicationMethod(ctx, jobDescription)
	if err != nil {
		return ApplicationMethod{Kind: MethodUnresolved}
	}
	return method
}
