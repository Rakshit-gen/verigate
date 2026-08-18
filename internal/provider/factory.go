package provider

import (
	"log"

	"github.com/rakshit-gen/verigate/internal/config"
)

// New selects and constructs the configured provider. If credentials for
// BOTH an OpenAI-compatible backend and Anthropic are present, it chains
// them behind a circuit-breaker Router — PROVIDER_NAME's choice goes
// first, the other becomes the automatic fallback. With only one
// credential set configured (the common case), it returns that single
// provider directly, unchanged from before this existed.
func New(cfg config.Config, nameSuffix string) Provider {
	primary := buildSingle(cfg, cfg.ProviderName, nameSuffix)

	var fallbackName string
	switch {
	case cfg.ProviderName != "anthropic" && cfg.AnthropicAPIKey != "":
		fallbackName = "anthropic"
	case cfg.ProviderName == "anthropic" && cfg.OpenAIAPIKey != "":
		fallbackName = "openai"
	default:
		return primary // no second credential set configured — nothing to chain
	}

	fallback := buildSingle(cfg, fallbackName, nameSuffix)
	log.Printf("provider chain%s: %s -> %s (circuit breaker: %d failures opens, %s cooldown)",
		nameSuffix, cfg.ProviderName, fallbackName, cfg.FallbackFailureThreshold, cfg.FallbackCooldown)

	return NewRouter(
		RouterEntry{Provider: primary, FailureThreshold: cfg.FallbackFailureThreshold, Cooldown: cfg.FallbackCooldown},
		RouterEntry{Provider: fallback, FailureThreshold: cfg.FallbackFailureThreshold, Cooldown: cfg.FallbackCooldown},
	)
}

func buildSingle(cfg config.Config, name, nameSuffix string) Provider {
	if name == "anthropic" {
		return NewAnthropic(cfg.AnthropicAPIKey, cfg.AnthropicBaseURL)
	}
	return NewOpenAICompat(name+nameSuffix, cfg.OpenAIBaseURL, cfg.OpenAIAPIKey)
}
