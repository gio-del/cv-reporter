package claude

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

// fakeAnthropicServer returns an httptest.Server that answers every
// /v1/messages call with body (a canned Messages API response), regardless
// of the request — enough to test what Client does with a response's
// usage, without needing a real API key or network access.
func fakeAnthropicServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

const selectAndRewriteResponse = `{
	"id": "msg_1",
	"type": "message",
	"role": "assistant",
	"model": "claude-sonnet-5",
	"stop_reason": "tool_use",
	"content": [
		{
			"type": "tool_use",
			"id": "toolu_1",
			"name": "select_and_rewrite",
			"input": {"entries": []}
		}
	],
	"usage": {
		"input_tokens": 1000,
		"output_tokens": 200,
		"cache_creation_input_tokens": 0,
		"cache_read_input_tokens": 0
	}
}`

func TestSelectAndRewrite_RecordsUsage(t *testing.T) {
	server := fakeAnthropicServer(t, selectAndRewriteResponse)
	c := NewWithOptions(option.WithBaseURL(server.URL), option.WithAPIKey("test-key"))

	_, err := c.SelectAndRewrite(context.Background(), generation.SelectionRequest{JobDescription: "a role"})
	if err != nil {
		t.Fatalf("SelectAndRewrite() error = %v", err)
	}

	calls := c.DrainUsage()
	if len(calls) != 1 {
		t.Fatalf("DrainUsage() = %d calls, want 1", len(calls))
	}
	got := calls[0]
	if got.CallType != "selection_rewrite" {
		t.Errorf("CallType = %q, want %q", got.CallType, "selection_rewrite")
	}
	if got.InputTokens != 1000 || got.OutputTokens != 200 {
		t.Errorf("tokens = %d/%d, want 1000/200", got.InputTokens, got.OutputTokens)
	}
	if got.EstimatedCostUSD <= 0 {
		t.Errorf("EstimatedCostUSD = %v, want > 0", got.EstimatedCostUSD)
	}

	// Draining again must not resurrect the same call.
	if again := c.DrainUsage(); len(again) != 0 {
		t.Errorf("second DrainUsage() = %+v, want empty", again)
	}
}

func TestDrainUsage_NothingRecordedYet_ReturnsEmpty(t *testing.T) {
	c := New()
	if got := c.DrainUsage(); len(got) != 0 {
		t.Errorf("DrainUsage() on a fresh Client = %+v, want empty", got)
	}
}
