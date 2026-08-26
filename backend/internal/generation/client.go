package generation

import "context"

// Client is the seam through which the generation service asks an LLM to
// do Selection+Rewrite, Cover Letter drafting, and RAL Range estimation.
// The real implementation calls the Claude API directly (ADR-0005); tests
// provide a fake so Claude API calls are mocked/stubbed at this boundary,
// per the PRD's Testing Decisions.
type Client interface {
	SelectAndRewrite(ctx context.Context, req SelectionRequest) (SelectionResult, error)
}
