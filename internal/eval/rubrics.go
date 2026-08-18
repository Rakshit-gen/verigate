package eval

// Rubric is a named grading prompt. The judge model is asked to score the
// (prompt, response) pair against it and return strict JSON so the worker
// can parse a number without another round of prompt-fiddling at 1am.
type Rubric struct {
	Name         string
	SystemPrompt string
}

var Rubrics = []Rubric{
	{
		Name: "groundedness",
		SystemPrompt: `You are a strict evaluator. You will be shown a user prompt and an AI
assistant's response. Score how grounded and non-fabricated the response is
on a 0.00-1.00 scale, where 1.00 means fully accurate and directly
supported by what a careful, honest answer would say, and 0.00 means the
response is fabricated, off-topic, or nonsensical.

Respond with ONLY a JSON object, no other text:
{"score": <number between 0 and 1>, "reasoning": "<one short sentence>"}`,
	},
	{
		Name: "format_compliance",
		SystemPrompt: `You are a strict evaluator. You will be shown a user prompt and an AI
assistant's response. Score whether the response is well-formed, on-topic,
and reasonably concise on a 0.00-1.00 scale. A response that is empty,
garbled, repeats itself, or completely ignores the question scores near 0.

Respond with ONLY a JSON object, no other text:
{"score": <number between 0 and 1>, "reasoning": "<one short sentence>"}`,
	},
}

// ToolCallRubric grades whether an assistant's tool call(s) were an
// appropriate response to the user's request — did it call the RIGHT
// tool, with sensible arguments, not just "was the text good" (which is
// meaningless for a turn that's a function call, not prose). Run
// separately from Rubrics, and only when a response actually contains
// tool calls — see eval/worker.go.
//
// Scoped honestly: this grades plausibility from the judge's general
// knowledge of what a tool name/arguments ought to look like for a given
// request. It does NOT have access to the tool's formal JSON schema
// (Verigate doesn't currently capture the `tools` array from the original
// request), which would make grading strictly more accurate — a real,
// tracked next step, not a hidden limitation.
var ToolCallRubric = Rubric{
	Name: "tool_call_correctness",
	SystemPrompt: `You are a strict evaluator of AI agent tool use. You will be shown a
user's request and the tool call(s) an AI assistant made in response —
the tool/function name(s) and arguments, as raw JSON. Score on a
0.00-1.00 scale whether calling this tool, with these arguments, is an
appropriate and sensible response to the user's request. A tool call that
matches the wrong intent, has clearly wrong or malformed arguments, or
that a user's request didn't call for at all scores near 0. A tool call
that plausibly matches what a helpful assistant would do scores near 1,
even without seeing the tool's formal schema — judge by what the tool
name and arguments would reasonably do.

Respond with ONLY a JSON object, no other text:
{"score": <number between 0 and 1>, "reasoning": "<one short sentence>"}`,
}
