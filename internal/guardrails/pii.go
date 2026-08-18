// Package guardrails protects what Verigate itself stores and logs — it
// does NOT filter or block traffic to/from the actual LLM provider. That
// distinction matters: redacting a prompt before forwarding it to the
// model would silently corrupt the request the caller asked for. What
// guardrails protects is Verigate's own database (requests/evals rows),
// which is a real, separate risk surface — every prompt a user sends
// passes through and is logged by this gateway.
package guardrails

import "regexp"

// piiPatterns are deliberately conservative (favor missing something over
// mangling normal text) — this is a best-effort logging safeguard, not a
// compliance-grade DLP system. Order matters: more specific patterns
// (credit card, API keys) run before broader ones so a match isn't
// partially consumed by a looser pattern first.
var piiPatterns = []struct {
	name    string
	pattern *regexp.Regexp
	replace string
}{
	{"email", regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`), "[REDACTED_EMAIL]"},
	// Grouped 4-4-4-4 (the format people actually type/paste cards in) or a
	// clean unbroken 15-16 digit run (Amex/Visa/Mastercard length) — NOT a
	// loose "13-16 digits with optional separators anywhere," which would
	// false-positive on order numbers, timestamps, and other long IDs.
	{"credit_card", regexp.MustCompile(`\b(?:\d{4}[ \-]){3}\d{4}\b|\b\d{15,16}\b`), "[REDACTED_CARD]"},
	{"ssn", regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), "[REDACTED_SSN]"},
	{"phone", regexp.MustCompile(`\b(?:\+?1[ \-]?)?\(?\d{3}\)?[ \-]?\d{3}[ \-]?\d{4}\b`), "[REDACTED_PHONE]"},
	// Common API key prefixes across major providers — catches a user
	// accidentally pasting a real secret into a prompt, which is a
	// realistic and higher-stakes leak than most PII categories here.
	{"api_key", regexp.MustCompile(`\b(?:sk-[a-zA-Z0-9]{10,}|sk-ant-[a-zA-Z0-9\-]{10,}|gsk_[a-zA-Z0-9]{10,}|AKIA[0-9A-Z]{16})\b`), "[REDACTED_API_KEY]"},
}

// RedactPII replaces recognizable PII/secrets in text with typed
// placeholders and reports whether anything was redacted, so callers can
// record that fact (e.g. a `pii_redacted` column) without needing to diff
// the before/after text themselves.
func RedactPII(text string) (redacted string, found bool) {
	redacted = text
	for _, p := range piiPatterns {
		if p.pattern.MatchString(redacted) {
			redacted = p.pattern.ReplaceAllString(redacted, p.replace)
			found = true
		}
	}
	return redacted, found
}
