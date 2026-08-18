package otelx

import "go.opentelemetry.io/otel/attribute"

// GenAI semantic-convention attribute keys, per the OpenTelemetry GenAI SIG
// spec (CNCF-graduated 2026, attribute names still marked experimental —
// hand-declared here rather than pulled from otel's semconv package
// because that package hasn't stabilized a GenAI section yet). Source of
// truth: https://opentelemetry.io/docs/specs/semconv/gen-ai/
const (
	AttrGenAISystem            = attribute.Key("gen_ai.system")         // e.g. "openai", "groq"
	AttrGenAIOperationName     = attribute.Key("gen_ai.operation.name") // e.g. "chat"
	AttrGenAIRequestModel      = attribute.Key("gen_ai.request.model")
	AttrGenAIResponseModel     = attribute.Key("gen_ai.response.model")
	AttrGenAIUsageInputTokens  = attribute.Key("gen_ai.usage.input_tokens")
	AttrGenAIUsageOutputTokens = attribute.Key("gen_ai.usage.output_tokens")

	// Verigate-specific attributes. Not part of any spec — namespaced under
	// verigate.* so they're unambiguously an extension, not a claim that
	// these are standardized.
	AttrVerigateCacheHit   = attribute.Key("verigate.cache.hit")
	AttrVerigateRequestID  = attribute.Key("verigate.request.id")
	AttrVerigateCostUSD    = attribute.Key("verigate.cost.usd")
	AttrVerigateEvalRubric = attribute.Key("verigate.eval.rubric")
	AttrVerigateEvalScore  = attribute.Key("verigate.eval.score")
	AttrVerigateStream     = attribute.Key("verigate.request.stream")
	AttrVerigateCacheType  = attribute.Key("verigate.cache.type") // "exact" | "semantic" | "miss"
)

// TokenDirection labels the gen_ai.token.type attribute on token-count
// metrics, matching the spec's split between input and output tokens.
type TokenDirection string

const (
	TokenInput  TokenDirection = "input"
	TokenOutput TokenDirection = "output"
)

const AttrGenAITokenType = attribute.Key("gen_ai.token.type")
