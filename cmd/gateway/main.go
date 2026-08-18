package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/rakshit-gen/verigate/internal/cache"
	"github.com/rakshit-gen/verigate/internal/config"
	"github.com/rakshit-gen/verigate/internal/embeddings"
	"github.com/rakshit-gen/verigate/internal/eval"
	"github.com/rakshit-gen/verigate/internal/otelx"
	"github.com/rakshit-gen/verigate/internal/provider"
	"github.com/rakshit-gen/verigate/internal/router"
	"github.com/rakshit-gen/verigate/internal/store"
)

func main() {
	_ = godotenv.Load() // fine if .env doesn't exist; real env vars still work

	cfg := config.Load()
	if cfg.ProviderName == "anthropic" && cfg.AnthropicAPIKey == "" {
		log.Println("WARNING: ANTHROPIC_API_KEY is not set — chat completions will fail until it is")
	} else if cfg.ProviderName != "anthropic" && cfg.OpenAIAPIKey == "" {
		log.Println("WARNING: OPENAI_API_KEY is not set — chat completions will fail until it is")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer st.Close()

	rc := cache.New(cfg.RedisAddr)
	if err := rc.Ping(ctx); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	if cfg.EmbeddingAPIKey != "" {
		embedder := embeddings.NewOpenAICompat(cfg.EmbeddingAPIKey, cfg.EmbeddingBaseURL, cfg.EmbeddingModel)
		rc.EnableSemanticCache(embedder, cfg.SemanticCacheThreshold, cfg.SemanticCacheMaxEntries)
		log.Printf("semantic cache: enabled (model=%s, threshold=%.2f, max_entries=%d)",
			cfg.EmbeddingModel, cfg.SemanticCacheThreshold, cfg.SemanticCacheMaxEntries)
	} else {
		log.Println("semantic cache: disabled (no EMBEDDING_API_KEY configured) — exact-match caching only")
	}

	otelProviders, err := otelx.Init(ctx, "verigate")
	if err != nil {
		log.Fatalf("failed to init opentelemetry: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := otelProviders.Shutdown(shutdownCtx); err != nil {
			log.Printf("otel shutdown error: %v", err)
		}
	}()
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		log.Printf("otel: exporting to %s", endpoint)
	} else {
		log.Printf("otel: no OTEL_EXPORTER_OTLP_ENDPOINT set — spans/metrics print to stdout")
	}

	chatProvider := provider.New(cfg, "")
	judgeProvider := provider.New(cfg, "-judge")
	judge := eval.NewJudge(judgeProvider, cfg.JudgeModel)

	sampler := eval.NewSampler(cfg.EvalSampleRate, st, judge, otelProviders)
	sampler.StartWorkers(ctx, 4)

	handler := router.New(router.Deps{
		Cfg:      cfg,
		Store:    st,
		Cache:    rc,
		Provider: chatProvider,
		Sampler:  sampler,
		Otel:     otelProviders,
	})

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: handler}

	go func() {
		log.Printf("verigate listening on :%s (sample rate %.0f%%, regression threshold %.2f)",
			cfg.Port, cfg.EvalSampleRate*100, cfg.RegressionMinScore)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
