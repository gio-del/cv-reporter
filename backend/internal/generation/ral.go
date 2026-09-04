package generation

import (
	"regexp"
	"strconv"
	"strings"
)

// RALSource labels where a RAL Range came from, per CONTEXT.md's RAL Range
// entry: never let the FE mistake an estimate for a fact.
type RALSource string

const (
	RALSourceStated    RALSource = "stated"
	RALSourceEstimated RALSource = "estimated"
	RALSourceNA        RALSource = "n/a"
	// RALSourceUnresolved means resolution couldn't even be attempted
	// (client.EstimateRAL returned an error) — distinct from RALSourceNA,
	// which means research ran and genuinely found nothing.
	RALSourceUnresolved RALSource = "unresolved"
)

// RALRange is the gross annual salary range for a Job Listing. Min/Max are
// nil when Source is RALSourceNA or RALSourceUnresolved.
type RALRange struct {
	Min      *int      `json:"min,omitempty"`
	Max      *int      `json:"max,omitempty"`
	Currency string    `json:"currency,omitempty"`
	Source   RALSource `json:"source"`
}

var (
	// ralKeywordRe matches a line naming pay explicitly, so a plain number
	// range (years of experience, a date range) isn't mistaken for salary.
	ralKeywordRe = regexp.MustCompile(`(?i)\b(RAL|salary|compensation|gross annual|annual gross|pay range)\b`)

	currencySymbolRe = regexp.MustCompile(`(€|\$|£)`)
	currencyCodeRe   = regexp.MustCompile(`(?i)\b(EUR|USD|GBP)\b`)

	// numberToken: digits with optional thousand separators, optional "k"
	// (thousands) suffix. currencyPrefix optionally absorbs a currency
	// symbol/code directly before a number (e.g. "€45,000"), since it can
	// appear before either or both numbers in a range.
	numberToken   = `[\d]{1,3}(?:[.,]\d{3})*\s*(?:k)?`
	currencyPrefx = `(?:€|\$|£|EUR|USD|GBP)?\s*`
	rangeRe       = regexp.MustCompile(`(?i)` + currencyPrefx + `(` + numberToken + `)\s*(?:-|–|—|to)\s*` + currencyPrefx + `(` + numberToken + `)`)
	singleRe      = regexp.MustCompile(`(?i)` + currencyPrefx + `(` + numberToken + `)`)
)

// ParseStatedRAL looks for a salary figure or range stated directly in a
// Job Description's text, per the PRD's RAL Range lookup: "parse the Job
// Description text first". It reports ok=false if it can't find one
// confidently — a bare number range (years of experience, dates) is not
// treated as a salary without a currency symbol/code or a pay keyword
// (RAL, salary, compensation, ...) on the same line.
func ParseStatedRAL(jobDescription string) (RALRange, bool) {
	for _, line := range strings.Split(jobDescription, "\n") {
		currency := currencyFrom(line)
		if currency == "" && !ralKeywordRe.MatchString(line) {
			continue
		}
		if currency == "" {
			currency = "EUR" // RAL is an Italian-market term; default accordingly.
		}

		if min, max, ok := parseRange(line); ok {
			return RALRange{Min: &min, Max: &max, Currency: currency, Source: RALSourceStated}, true
		}
		if v, ok := parseSingle(line); ok {
			return RALRange{Min: &v, Max: &v, Currency: currency, Source: RALSourceStated}, true
		}
	}
	return RALRange{}, false
}

// parseRange looks for "<num> - <num>" in line. A "k" suffix on the second
// number implies the same unit for the first if the first looks like
// shorthand (e.g. "40-50k" means 40k-50k, not 40 and 50000).
func parseRange(line string) (min, max int, ok bool) {
	m := rangeRe.FindStringSubmatch(line)
	if m == nil {
		return 0, 0, false
	}
	raw1, k1 := splitK(m[1])
	raw2, k2 := splitK(m[2])
	v1, ok1 := parseAmount(raw1)
	v2, ok2 := parseAmount(raw2)
	if !ok1 || !ok2 {
		return 0, 0, false
	}
	if k2 && !k1 && v1 < 1000 {
		k1 = true
	}
	if k1 {
		v1 *= 1000
	}
	if k2 {
		v2 *= 1000
	}
	if v1 > v2 {
		v1, v2 = v2, v1
	}
	return v1, v2, true
}

func parseSingle(line string) (int, bool) {
	m := singleRe.FindStringSubmatch(line)
	if m == nil {
		return 0, false
	}
	raw, k := splitK(m[1])
	v, ok := parseAmount(raw)
	if !ok {
		return 0, false
	}
	if k {
		v *= 1000
	}
	return v, true
}

func splitK(token string) (raw string, hasK bool) {
	token = strings.TrimSpace(token)
	if strings.HasSuffix(strings.ToLower(token), "k") {
		return strings.TrimSpace(token[:len(token)-1]), true
	}
	return token, false
}

func currencyFrom(line string) string {
	if currencySymbolRe.MatchString(line) {
		switch currencySymbolRe.FindString(line) {
		case "€":
			return "EUR"
		case "$":
			return "USD"
		case "£":
			return "GBP"
		}
	}
	if m := currencyCodeRe.FindString(line); m != "" {
		return strings.ToUpper(m)
	}
	return ""
}

// parseAmount turns a matched number token ("45,000", "90.000") into a
// whole-currency-unit integer. Any "k" suffix must already be stripped by
// the caller (see splitK) since it scales the parsed value, not the token.
func parseAmount(token string) (int, bool) {
	cleaned := strings.NewReplacer(",", "", ".", "").Replace(strings.TrimSpace(token))
	if cleaned == "" {
		return 0, false
	}
	v, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0, false
	}
	return v, true
}
