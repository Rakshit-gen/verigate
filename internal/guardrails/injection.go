package guardrails

import (
	"regexp"
	"strings"
)

// injectionSignal is one heuristic pattern with a weight reflecting how
// specific/damning a match is: a phrase that's almost never legitimate
// ("ignore all previous instructions") weighs more than one with
// plausible innocent uses ("act as a"), so a single strong signal can
// still cross the flag threshold on its own while weak signals need to
// stack up.
type injectionSignal struct {
	pattern *regexp.Regexp
	weight  float64
}

var injectionSignals = []injectionSignal{
	{regexp.MustCompile(`(?i)ignore (all |any )?(previous|prior|above|earlier) (instructions?|prompts?|rules?)`), 0.6},
	{regexp.MustCompile(`(?i)disregard (the |your |all )?(system prompt|previous|instructions?)`), 0.6},
	{regexp.MustCompile(`(?i)(reveal|print|show|repeat|tell me) (your |the )?(system prompt|instructions?)`), 0.5},
	{regexp.MustCompile(`(?i)you are (now |no longer )?(in )?(developer|debug|dan|jailbreak|unrestricted) mode`), 0.7},
	{regexp.MustCompile(`(?i)pretend (you have|to have) no (restrictions|rules|guidelines)`), 0.5},
	{regexp.MustCompile(`(?i)act as (if you (are|were)|an? )?(unrestricted|uncensored|unfiltered)`), 0.4},
	{regexp.MustCompile(`(?i)from now on,? (you|ignore|forget)`), 0.3},
	{regexp.MustCompile(`(?i)this is (a |an )?(test|hypothetical) (scenario|situation)( where| in which)?.{0,30}(no rules|anything goes|no restrictions)`), 0.4},
}

// PromptInjectionScore returns a 0-1 heuristic risk score for a piece of
// text (typically a user prompt) plus the names of whatever patterns
// matched, for logging/debugging. This is intentionally a heuristic
// keyword/pattern scorer, not a classifier — a determined attacker can
// phrase around any fixed pattern list, and the point here is a cheap
// first-pass signal worth logging and alerting on, not a hard block. A
// judge-model-based classifier would catch more but costs a real API call
// per request; this costs nothing and catches the common, lazy cases.
func PromptInjectionScore(text string) (score float64, matchedSignals []string) {
	lower := strings.ToLower(text)
	for _, sig := range injectionSignals {
		if sig.pattern.MatchString(lower) {
			score += sig.weight
			matchedSignals = append(matchedSignals, sig.pattern.String())
		}
	}
	if score > 1.0 {
		score = 1.0
	}
	return score, matchedSignals
}
