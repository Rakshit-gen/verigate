package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port           string
	DatabaseURL    string
	RedisAddr      string
	VerigateAPIKey string

	ProviderName  string // label stored on every request row — "openai", "groq", "ollama", "anthropic"
	OpenAIAPIKey  string
	OpenAIBaseURL string // any OpenAI-compatible endpoint: OpenAI, Groq, local Ollama/vLLM
	JudgeModel    string
	ChatModel     string

	// Used only when PROVIDER_NAME=anthropic. Anthropic's Messages API is
	// not OpenAI-wire-compatible, so it's a genuinely separate adapter
	// rather than another OPENAI_BASE_URL value.
	AnthropicAPIKey  string
	AnthropicBaseURL string

	// If both an OpenAI-compatible key AND an Anthropic key are present,
	// cmd/gateway chains them behind a circuit breaker — whichever
	// PROVIDER_NAME points to goes first, the other becomes the fallback.
	// These tune that breaker; see internal/provider/circuitrouter.go.
	FallbackFailureThreshold int
	FallbackCooldown         time.Duration

	EvalSampleRate float64 // 0.0-1.0, fraction of requests sent to the judge

	RegressionWindow         int     // recent-window size for the eval rolling average
	RegressionBaselineWindow int     // size of the baseline window compared against
	RegressionZThreshold     float64 // z-score above which a drop counts as a regression
	RegressionMinScore       float64 // fixed floor used only during cold start (see RegressionSummary)

	// Semantic caching is opt-in: it needs a real embeddings-capable key,
	// which not every provider (Groq, for one) offers. Empty
	// EmbeddingAPIKey disables it and the cache silently stays exact-match
	// only — see internal/cache for the fallback.
	EmbeddingAPIKey         string
	EmbeddingBaseURL        string
	EmbeddingModel          string
	SemanticCacheThreshold  float64
	SemanticCacheMaxEntries int
}

func Load() Config {
	cfg := Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://localhost:5432/verigate?sslmode=disable"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		VerigateAPIKey: getEnv("VERIGATE_API_KEY", "dev-local-key"),

		ProviderName:  getEnv("PROVIDER_NAME", "openai"),
		OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		JudgeModel:    getEnv("JUDGE_MODEL", "gpt-4o-mini"),
		ChatModel:     getEnv("CHAT_MODEL_DEFAULT", "gpt-4o-mini"),

		AnthropicAPIKey:  getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicBaseURL: getEnv("ANTHROPIC_BASE_URL", "https://api.anthropic.com/v1"),

		FallbackFailureThreshold: getEnvInt("FALLBACK_FAILURE_THRESHOLD", 3),
		FallbackCooldown:         time.Duration(getEnvInt("FALLBACK_COOLDOWN_SECONDS", 30)) * time.Second,

		EvalSampleRate: getEnvFloat("EVAL_SAMPLE_RATE", 0.3),

		RegressionWindow:         getEnvInt("REGRESSION_WINDOW", 20),
		RegressionBaselineWindow: getEnvInt("REGRESSION_BASELINE_WINDOW", 100),
		RegressionZThreshold:     getEnvFloat("REGRESSION_Z_THRESHOLD", 2.0),
		RegressionMinScore:       getEnvFloat("REGRESSION_MIN_SCORE", 0.6),

		EmbeddingAPIKey:         getEnv("EMBEDDING_API_KEY", ""),
		EmbeddingBaseURL:        getEnv("EMBEDDING_BASE_URL", "https://api.openai.com/v1"),
		EmbeddingModel:          getEnv("EMBEDDING_MODEL", "text-embedding-3-small"),
		SemanticCacheThreshold:  getEnvFloat("SEMANTIC_CACHE_THRESHOLD", 0.92),
		SemanticCacheMaxEntries: getEnvInt("SEMANTIC_CACHE_MAX_ENTRIES", 500),
	}

	// If no dedicated embedding key was set but the chat provider IS OpenAI
	// itself, reuse that key rather than making the user configure the same
	// credential twice — only do this when the provider is actually OpenAI,
	// since Groq/Anthropic keys don't work against OpenAI's embeddings API.
	if cfg.EmbeddingAPIKey == "" && cfg.ProviderName == "openai" {
		cfg.EmbeddingAPIKey = cfg.OpenAIAPIKey
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
