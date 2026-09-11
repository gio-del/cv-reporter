package tracking

import (
	"context"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// HTTPDoer is the minimal http.Client surface the Company Logo download
// depends on, so callers (including tests) can inject a fake instead of
// making a live network call — the same seam atsboard.HTTPDoer establishes
// for ATS board fetches. *http.Client satisfies this interface.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// maxLogoBytes caps how much of a logo response body Save will read, so a
// misbehaving or hostile response can't exhaust memory on a best-effort,
// never-blocks-the-save download (ADR-0013).
const maxLogoBytes = 5 << 20 // 5MB

// downloadLogoBestEffort fetches logoURL and writes it to jobsFullDir as
// "<slug><ext>", returning that filename — or "" on any failure (network
// error, non-200, empty body), per ADR-0013's "never blocks the save"
// requirement (story 11). A nil doer or empty logoURL is also a no-op.
func downloadLogoBestEffort(ctx context.Context, doer HTTPDoer, logoURL, jobsFullDir, slug string) string {
	if doer == nil || strings.TrimSpace(logoURL) == "" {
		return ""
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, logoURL, nil)
	if err != nil {
		return ""
	}
	resp, err := doer.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close() //nolint:errcheck // idiomatic response-body drain: the read is already done and there's nothing to do about a close failure
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLogoBytes))
	if err != nil || len(body) == 0 {
		return ""
	}

	ext, ok := imageExtension(resp.Header.Get("Content-Type"), body)
	if !ok {
		return ""
	}

	filename := slug + ext
	if err := os.WriteFile(filepath.Join(jobsFullDir, filename), body, 0o644); err != nil {
		return ""
	}
	return filename
}

// imageExtension picks a file extension for a downloaded logo from its
// response Content-Type, falling back to sniffing the bytes themselves
// (LinkedIn's CDN URLs don't reliably carry a file extension), and
// reports ok=false when the content isn't recognizable as an image at
// all — a Company Logo, not just any successfully-fetched URL.
func imageExtension(contentType string, body []byte) (ext string, ok bool) {
	mediaType, _, _ := mime.ParseMediaType(contentType) //nolint:errcheck // an unparseable Content-Type is handled by the DetectContentType fallback below
	if !strings.HasPrefix(mediaType, "image/") {
		mediaType = http.DetectContentType(body)
	}
	if !strings.HasPrefix(mediaType, "image/") {
		return "", false
	}
	if exts, err := mime.ExtensionsByType(mediaType); err == nil && len(exts) > 0 {
		return exts[0], true
	}
	return ".png", true
}
