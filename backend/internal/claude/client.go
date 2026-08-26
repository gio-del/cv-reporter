// Package claude is the real generation.Client implementation: it calls the
// Claude API directly, per ADR-0005. Every method forces a tool call so the
// response is structured, then decodes that tool call's input into the
// generation package's result types.
package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// Client wraps the Anthropic SDK client with the prompts and tool schemas
// the generation pipeline needs. It reads ANTHROPIC_API_KEY from the
// environment (via the SDK's default option), so constructing one never
// fails — only a call that actually reaches the API can, if the key is
// missing or invalid.
type Client struct {
	api   anthropic.Client
	model anthropic.Model
}

// New builds a Client using ANTHROPIC_API_KEY from the environment.
func New() *Client {
	return &Client{api: anthropic.NewClient(), model: anthropic.ModelClaudeSonnet5}
}

// NewWithOptions builds a Client with explicit SDK request options (tests
// pointing at a fake server, a non-default model, etc).
func NewWithOptions(opts ...option.RequestOption) *Client {
	return &Client{api: anthropic.NewClient(opts...), model: anthropic.ModelClaudeSonnet5}
}

var _ generation.Client = (*Client)(nil)

const selectAndRewriteSystemPrompt = `You are Tailoring a CV for one specific Job Description, following the "Tailor CV" process described below.

Selection: choose which Entries, and which of their bullets, are relevant to the Job Description, and in what order. The result must be trimmable enough to fit one page, so err on the side of cutting a marginal Entry or bullet rather than keeping everything.

Rewrite: adjust bullet phrasing to better match the Job Description's language and emphasis. Do not introduce facts, tools, or claims that aren't present in the source bullet — you may reword, never invent. For each bullet you keep, report its exact original text as "source" (verbatim, unmodified) and your reworded version as "rewritten". If you don't reword a bullet, "rewritten" must equal "source".

Call the select_and_rewrite tool with your result. Every "entryId" you return must be one of the candidate ids given to you. Every "sourceIndex" must be that bullet's position (0-based) in that Entry's bullet list, and "source" must match that bullet's text exactly.`

const selectAndRewriteToolName = "select_and_rewrite"

func selectAndRewriteTool() anthropic.ToolUnionParam {
	schema := anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"entries": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"entryId": map[string]any{"type": "string", "description": "Must match one of the candidate Entry ids."},
						"reason":  map[string]any{"type": "string", "description": "Why this Entry was selected, for Text Review."},
						"bullets": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"sourceIndex": map[string]any{"type": "integer"},
									"source":      map[string]any{"type": "string"},
									"rewritten":   map[string]any{"type": "string"},
								},
								"required": []string{"sourceIndex", "source", "rewritten"},
							},
						},
					},
					"required": []string{"entryId", "reason", "bullets"},
				},
			},
		},
		Required: []string{"entries"},
	}
	return anthropic.ToolUnionParamOfTool(schema, selectAndRewriteToolName)
}

// SelectAndRewrite asks Claude to select and rewrite Entries for req, via a
// forced call to the select_and_rewrite tool.
func (c *Client) SelectAndRewrite(ctx context.Context, req generation.SelectionRequest) (generation.SelectionResult, error) {
	candidates, err := json.Marshal(req.Candidates)
	if err != nil {
		return generation.SelectionResult{}, fmt.Errorf("marshaling candidates: %w", err)
	}

	userPrompt := fmt.Sprintf("Job Description:\n%s\n\nCandidate Entries (JSON):\n%s", req.JobDescription, candidates)

	message, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:      c.model,
		MaxTokens:  8192,
		System:     []anthropic.TextBlockParam{{Text: selectAndRewriteSystemPrompt}},
		Messages:   []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt))},
		Tools:      []anthropic.ToolUnionParam{selectAndRewriteTool()},
		ToolChoice: anthropic.ToolChoiceParamOfTool(selectAndRewriteToolName),
	})
	if err != nil {
		return generation.SelectionResult{}, fmt.Errorf("calling Claude API: %w", err)
	}

	for _, block := range message.Content {
		if block.Type != "tool_use" || block.Name != selectAndRewriteToolName {
			continue
		}
		var result generation.SelectionResult
		if err := json.Unmarshal(block.Input, &result); err != nil {
			return generation.SelectionResult{}, fmt.Errorf("decoding %s tool input: %w", selectAndRewriteToolName, err)
		}
		return result, nil
	}
	return generation.SelectionResult{}, fmt.Errorf("Claude response had no %s tool call", selectAndRewriteToolName)
}

const draftCoverLetterSystemPrompt = `You draft a Cover Letter for one specific Job Description.

If any Cover Letter Snippets are given, select and lightly adapt among them for this Job Description; list every Snippet id you drew from in "sourceSnippetIds". If none are given, or none fit, write fresh prose grounded only in the candidate Entries and the Job Description — leave "sourceSnippetIds" empty in that case.

Same constraint as Rewrite: do not invent facts, tools, employers, or claims absent from the candidate Entries or the Snippets you cite.

Call the draft_cover_letter tool with your result.`

const draftCoverLetterToolName = "draft_cover_letter"

func draftCoverLetterTool() anthropic.ToolUnionParam {
	schema := anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"body": map[string]any{"type": "string", "description": "The full Cover Letter prose."},
			"sourceSnippetIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Ids of any Cover Letter Snippets selected/adapted from. Empty if freshly generated.",
			},
		},
		Required: []string{"body"},
	}
	return anthropic.ToolUnionParamOfTool(schema, draftCoverLetterToolName)
}

// DraftCoverLetter asks Claude to draft a Cover Letter for req, via a
// forced call to the draft_cover_letter tool.
func (c *Client) DraftCoverLetter(ctx context.Context, req generation.CoverLetterRequest) (generation.CoverLetterResult, error) {
	candidates, err := json.Marshal(req.Candidates)
	if err != nil {
		return generation.CoverLetterResult{}, fmt.Errorf("marshaling candidates: %w", err)
	}
	snippets, err := json.Marshal(req.Snippets)
	if err != nil {
		return generation.CoverLetterResult{}, fmt.Errorf("marshaling snippets: %w", err)
	}

	userPrompt := fmt.Sprintf(
		"Job Description:\n%s\n\nCandidate Entries (JSON):\n%s\n\nCover Letter Snippets (JSON):\n%s",
		req.JobDescription, candidates, snippets,
	)

	message, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:      c.model,
		MaxTokens:  4096,
		System:     []anthropic.TextBlockParam{{Text: draftCoverLetterSystemPrompt}},
		Messages:   []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt))},
		Tools:      []anthropic.ToolUnionParam{draftCoverLetterTool()},
		ToolChoice: anthropic.ToolChoiceParamOfTool(draftCoverLetterToolName),
	})
	if err != nil {
		return generation.CoverLetterResult{}, fmt.Errorf("calling Claude API: %w", err)
	}

	for _, block := range message.Content {
		if block.Type != "tool_use" || block.Name != draftCoverLetterToolName {
			continue
		}
		var result generation.CoverLetterResult
		if err := json.Unmarshal(block.Input, &result); err != nil {
			return generation.CoverLetterResult{}, fmt.Errorf("decoding %s tool input: %w", draftCoverLetterToolName, err)
		}
		return result, nil
	}
	return generation.CoverLetterResult{}, fmt.Errorf("Claude response had no %s tool call", draftCoverLetterToolName)
}
