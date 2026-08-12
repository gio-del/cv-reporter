package masterdata

import (
	"bytes"
	"fmt"
	"strings"
)

// splitFrontmatter splits an Entry file's content into its YAML frontmatter
// and Markdown body, per the shape established by ADR-0003 (--- delimited
// YAML, then a Markdown body).
func splitFrontmatter(content []byte) (frontmatter, body []byte, err error) {
	trimmed := bytes.TrimLeft(content, "\n")
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return nil, nil, fmt.Errorf("missing frontmatter delimiter")
	}
	rest := trimmed[len("---"):]
	rest = bytes.TrimPrefix(rest, []byte("\r\n"))
	rest = bytes.TrimPrefix(rest, []byte("\n"))

	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		return nil, nil, fmt.Errorf("missing closing frontmatter delimiter")
	}
	frontmatter = rest[:idx]

	after := rest[idx+len("\n---"):]
	after = bytes.TrimPrefix(after, []byte("\r\n"))
	after = bytes.TrimPrefix(after, []byte("\n"))
	body = after
	return frontmatter, body, nil
}

// extractBullets reads the Markdown body's top-level "- " list items.
func extractBullets(body []byte) []string {
	var bullets []string
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			bullets = append(bullets, strings.TrimPrefix(trimmed, "- "))
		}
	}
	return bullets
}
