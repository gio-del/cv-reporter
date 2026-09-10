package generation

import "strings"

// DefaultLanguage is the fallback used whenever a detected/requested
// language falls outside SupportedLanguages, or none is given at all — the
// tool's original, English-only behavior (see issue #41's PRD).
const DefaultLanguage = "en"

// SupportedLanguages is the initial scope for the Tailored CV/Cover
// Letter's target language: English and Italian only, matching the two
// markets the domain already has real support for (RAL Range is
// Italian-market-specific; the tool's baseline is English).
var SupportedLanguages = map[string]bool{
	"en": true,
	"it": true,
}

// NormalizeLanguage maps code to a supported language, defaulting to
// DefaultLanguage for anything empty, unsupported, or out of scope, so a
// Job Description in an unrecognized or ambiguous language never blocks a
// Generation that would have succeeded before this feature existed.
func NormalizeLanguage(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if SupportedLanguages[code] {
		return code
	}
	return DefaultLanguage
}
