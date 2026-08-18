package guardrails

import (
	"strings"
	"testing"
)

func TestRedactPII(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantFound bool
		wantGone  string // substring that must NOT appear in the redacted output
	}{
		{"email", "contact me at jane.doe@example.com please", true, "jane.doe@example.com"},
		{"grouped credit card", "my card is 4242 4242 4242 4242", true, "4242 4242 4242 4242"},
		{"unbroken credit card", "card number 4242424242424242 exp 12/26", true, "4242424242424242"},
		{"ssn", "my ssn is 123-45-6789", true, "123-45-6789"},
		{"phone", "call me at 415-555-0132", true, "415-555-0132"},
		{"openai key", "here is my key sk-abcdefghij1234567890", true, "sk-abcdefghij1234567890"},
		{"groq key", "use gsk_abcdefghij1234567890 as the key", true, "gsk_abcdefghij1234567890"},
		{"clean text unaffected", "what's the capital of France?", false, ""},
		{"ordinary numbers not over-redacted", "I ordered 12345 units on order #98765", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			redacted, found := RedactPII(tc.input)
			if found != tc.wantFound {
				t.Errorf("RedactPII(%q) found=%v, want %v (redacted=%q)", tc.input, found, tc.wantFound, redacted)
			}
			if tc.wantGone != "" && strings.Contains(redacted, tc.wantGone) {
				t.Errorf("RedactPII(%q) = %q, expected %q to be redacted out", tc.input, redacted, tc.wantGone)
			}
			if !tc.wantFound && redacted != tc.input {
				t.Errorf("expected clean text to pass through unchanged, got %q", redacted)
			}
		})
	}
}
