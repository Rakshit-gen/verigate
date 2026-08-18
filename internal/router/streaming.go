package router

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/rakshit-gen/verigate/internal/otelx"
	"github.com/rakshit-gen/verigate/internal/provider"
	"github.com/rakshit-gen/verigate/internal/store"
)

// handleStreamingChat proxies a `stream: true` chat completion chunk-by-
// chunk as the upstream provider produces it, while still accumulating the
// full text so it can be logged, cached, and sampled for eval exactly like
// a buffered request — streaming clients get the same product guarantees,
// not a second-class path.
//
// Caching works both directions: a streaming request first checks the same
// exact-match/semantic cache a non-streaming request would (using the same
// normalized key — see cache.Key — so the two request styles share
// entries), and on a hit, synthesizes an SSE reply from the cached
// OpenAI-shaped completion instead of calling the provider at all. On a
// miss, the accumulated response gets cached in that same OpenAI-shaped
// format once the stream completes, so a LATER non-streaming request for
// the same prompt can hit it too.
func handleStreamingChat(ctx context.Context, d Deps, w http.ResponseWriter, span trace.Span, rawBody []byte, key, model, lastUserPrompt, tenantID string, start time.Time) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		span.SetStatus(codes.Error, "response writer does not support flushing")
		http.Error(w, `{"error":"streaming not supported by this server"}`, http.StatusInternalServerError)
		return
	}
	span.SetAttributes(otelx.AttrVerigateStream.Bool(true))

	metricAttrs := metric.WithAttributes(
		otelx.AttrGenAISystem.String(d.Provider.Name()),
		otelx.AttrGenAIRequestModel.String(model),
	)

	if cached, cacheType, hit := lookupCache(ctx, d.Cache, key, lastUserPrompt); hit {
		serveStreamFromCache(ctx, d, w, flusher, span, metricAttrs, cached, cacheType, model, lastUserPrompt, tenantID, start)
		return
	}
	span.SetAttributes(
		otelx.AttrVerigateCacheHit.Bool(false),
		otelx.AttrVerigateCacheType.String("miss"),
	)

	upstream, err := d.Provider.StreamChatCompletion(ctx, withUsageOnStream(rawBody))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Printf("provider stream error: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		gPrompt, _, piiRedacted, injScore := guardedFields(lastUserPrompt, "")
		d.Store.InsertRequest(ctx, store.RequestRecord{
			Provider: d.Provider.Name(), Model: model, Prompt: gPrompt,
			LatencyMS: int(time.Since(start).Milliseconds()), Status: "error",
			PIIRedacted: piiRedacted, InjectionScore: injScore, TenantID: tenantID,
		})
		return
	}
	defer upstream.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Verigate-Cache", "miss")
	w.WriteHeader(http.StatusOK)

	var content strings.Builder
	var tokensIn, tokensOut int
	var ttft time.Duration
	gotFirstToken := false

	reader := bufio.NewReader(upstream)
	for {
		line, readErr := reader.ReadString('\n')

		if len(line) > 0 {
			// Forward the exact bytes we received, immediately — this is a
			// proxy, not a re-encoder, and the client's SSE parser expects
			// OpenAI's own framing verbatim.
			w.Write([]byte(line))
			flusher.Flush()

			if payload, isData := parseSSEDataLine(line); isData && payload != "[DONE]" {
				if !gotFirstToken {
					ttft = time.Since(start)
					gotFirstToken = true
				}
				delta, usage := parseStreamChunk(payload)
				content.WriteString(delta)
				if usage != nil {
					tokensIn, tokensOut = usage.PromptTokens, usage.CompletionTokens
				}
			}
		}

		if readErr != nil {
			if readErr != io.EOF {
				span.RecordError(readErr)
			}
			break
		}
	}

	latency := time.Since(start)
	cost := provider.EstimateCost(model, tokensIn, tokensOut)

	span.SetAttributes(
		otelx.AttrGenAIUsageInputTokens.Int(tokensIn),
		otelx.AttrGenAIUsageOutputTokens.Int(tokensOut),
		otelx.AttrVerigateCostUSD.Float64(cost),
	)
	d.Otel.RequestCounter.Add(ctx, 1, metricAttrs)
	d.Otel.LatencyHist.Record(ctx, float64(latency.Milliseconds()), metricAttrs)
	if gotFirstToken {
		d.Otel.TTFTHist.Record(ctx, ttft.Seconds(), metricAttrs)
	}
	d.Otel.TokenCounter.Add(ctx, int64(tokensIn), metric.WithAttributes(
		otelx.AttrGenAISystem.String(d.Provider.Name()), otelx.AttrGenAITokenType.String(string(otelx.TokenInput))))
	d.Otel.TokenCounter.Add(ctx, int64(tokensOut), metric.WithAttributes(
		otelx.AttrGenAISystem.String(d.Provider.Name()), otelx.AttrGenAITokenType.String(string(otelx.TokenOutput))))

	if content.Len() > 0 {
		cacheable := buildCacheableCompletion(model, content.String(), tokensIn, tokensOut)
		if err := d.Cache.Set(ctx, key, cacheable); err != nil {
			log.Printf("cache set failed for streamed response: %v", err)
		} else {
			d.Cache.IndexForSemanticLookup(ctx, lastUserPrompt, key)
		}
	}

	gPrompt, gResponse, piiRedacted, injScore := guardedFields(lastUserPrompt, content.String())
	id, err := d.Store.InsertRequest(ctx, store.RequestRecord{
		Provider:       d.Provider.Name(),
		Model:          model,
		Prompt:         gPrompt,
		Response:       gResponse,
		LatencyMS:      int(latency.Milliseconds()),
		CacheHit:       false,
		TokensIn:       tokensIn,
		TokensOut:      tokensOut,
		CostUSD:        cost,
		Status:         "ok",
		PIIRedacted:    piiRedacted,
		InjectionScore: injScore,
		TenantID:       tenantID,
	})
	if err != nil {
		log.Printf("failed to log streamed request: %v", err)
		return
	}
	span.SetAttributes(otelx.AttrVerigateRequestID.String(id))
	d.Sampler.MaybeSample(id)
}

// serveStreamFromCache synthesizes an SSE reply from a cached OpenAI-shaped
// chat.completion — one content chunk plus a final usage chunk plus
// [DONE], not a token-by-token replay. That's a deliberate, documented
// trade-off (see docs/ARCHITECTURE.md): a cache hit answers near-instantly
// either way, and reconstructing the ORIGINAL chunk timing/boundaries
// would add real complexity for no observable benefit to the caller, who
// asked for `stream: true` to get incremental delivery of a slow
// generation — a cache hit isn't slow.
func serveStreamFromCache(ctx context.Context, d Deps, w http.ResponseWriter, flusher http.Flusher, span trace.Span, metricAttrs metric.MeasurementOption, cached []byte, cacheType, model, lastUserPrompt, tenantID string, start time.Time) {
	latency := time.Since(start)
	span.SetAttributes(
		otelx.AttrVerigateCacheHit.Bool(true),
		otelx.AttrVerigateCacheType.String(cacheType),
	)
	d.Otel.RequestCounter.Add(ctx, 1, metricAttrs)
	d.Otel.LatencyHist.Record(ctx, float64(latency.Milliseconds()), metricAttrs)
	d.Otel.TTFTHist.Record(ctx, latency.Seconds(), metricAttrs) // a cache hit's "first token" is effectively the whole response

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Verigate-Cache", cacheType)
	w.WriteHeader(http.StatusOK)

	content := extractContentFromRaw(cached)
	tokensIn, tokensOut := extractUsageFromRaw(cached)

	writeSSEChunk(w, flusher, map[string]any{
		"id": "verigate-cached", "object": "chat.completion.chunk", "created": time.Now().Unix(), "model": model,
		"choices": []map[string]any{{"index": 0, "delta": map[string]string{"content": content}, "finish_reason": nil}},
	})
	writeSSEChunk(w, flusher, map[string]any{
		"id": "verigate-cached", "object": "chat.completion.chunk", "created": time.Now().Unix(), "model": model,
		"choices": []map[string]any{{"index": 0, "delta": map[string]string{}, "finish_reason": "stop"}},
		"usage": map[string]int{
			"prompt_tokens": tokensIn, "completion_tokens": tokensOut, "total_tokens": tokensIn + tokensOut,
		},
	})
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()

	gPrompt, gResponse, piiRedacted, injScore := guardedFields(lastUserPrompt, content)
	id, err := d.Store.InsertRequest(ctx, store.RequestRecord{
		Provider:       d.Provider.Name(),
		Model:          model,
		Prompt:         gPrompt,
		Response:       gResponse,
		LatencyMS:      int(latency.Milliseconds()),
		CacheHit:       true,
		TokensIn:       tokensIn,
		TokensOut:      tokensOut,
		Status:         "ok",
		PIIRedacted:    piiRedacted,
		InjectionScore: injScore,
		TenantID:       tenantID,
	})
	if err != nil {
		log.Printf("failed to log cached streamed request: %v", err)
		return
	}
	span.SetAttributes(otelx.AttrVerigateRequestID.String(id))
	d.Sampler.MaybeSample(id)
}

func writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, chunk map[string]any) {
	payload, err := json.Marshal(chunk)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", payload)
	flusher.Flush()
}

// buildCacheableCompletion synthesizes an OpenAI-shaped chat.completion
// JSON from an accumulated streamed response, so streaming and
// non-streaming requests store (and can hit) the exact same cache value
// format regardless of which path produced it.
func buildCacheableCompletion(model, content string, tokensIn, tokensOut int) []byte {
	body := map[string]any{
		"id": "verigate-cached", "object": "chat.completion", "created": time.Now().Unix(), "model": model,
		"choices": []map[string]any{
			{"index": 0, "message": map[string]string{"role": "assistant", "content": content}, "finish_reason": "stop"},
		},
		"usage": map[string]int{
			"prompt_tokens": tokensIn, "completion_tokens": tokensOut, "total_tokens": tokensIn + tokensOut,
		},
	}
	out, _ := json.Marshal(body)
	return out
}

// parseSSEDataLine strips the "data: " prefix OpenAI-shaped SSE streams use
// and reports whether the line was a data line at all (blank lines are
// legitimate SSE frame separators, not something to parse).
func parseSSEDataLine(line string) (payload string, ok bool) {
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "data:")), true
}

type streamUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

func parseStreamChunk(payload string) (delta string, usage *streamUsage) {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
		Usage *streamUsage `json:"usage"`
	}
	if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
		return "", nil
	}
	if len(chunk.Choices) > 0 {
		delta = chunk.Choices[0].Delta.Content
	}
	return delta, chunk.Usage
}

// withUsageOnStream opts the upstream request into stream_options.
// include_usage (an OpenAI-spec field standard SDKs already know to ignore
// when iterating deltas) so Verigate gets real token counts for cost
// tracking on streamed responses too, without the caller having to know to
// ask for it. Fails open — a body Verigate can't parse is forwarded as-is
// rather than dropped.
func withUsageOnStream(rawBody []byte) []byte {
	var m map[string]any
	if err := json.Unmarshal(rawBody, &m); err != nil {
		return rawBody
	}
	if _, exists := m["stream_options"]; !exists {
		m["stream_options"] = map[string]bool{"include_usage": true}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return rawBody
	}
	return out
}
