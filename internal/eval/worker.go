package eval

import (
	"context"
	"log"
	"math/rand"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/rakshit-gen/verigate/internal/otelx"
	"github.com/rakshit-gen/verigate/internal/store"
)

// Sampler decides which requests get graded and hands them to a pool of
// worker goroutines. It's deliberately an in-memory channel rather than a
// real queue (Kafka/BullMQ) — tonight's traffic volume doesn't need one,
// and swapping the channel for a queue later is a small, contained change
// because nothing outside this file knows the difference.
type Sampler struct {
	rate  float64
	queue chan string
	store *store.Store
	judge *Judge
	otel  *otelx.Providers
}

func NewSampler(rate float64, s *store.Store, j *Judge, o *otelx.Providers) *Sampler {
	return &Sampler{
		rate:  rate,
		queue: make(chan string, 256),
		store: s,
		judge: j,
		otel:  o,
	}
}

// MaybeSample rolls the sample rate and enqueues the request for grading.
// Called once per logged request from the request-handling path — it never
// blocks the caller (the channel is buffered) and never does I/O itself.
func (s *Sampler) MaybeSample(requestID string) {
	if rand.Float64() > s.rate {
		return
	}
	select {
	case s.queue <- requestID:
	default:
		log.Printf("eval sampler: queue full, dropping sample for %s", requestID)
	}
}

// StartWorkers spawns n goroutines that consume the sample queue until ctx
// is cancelled. Each worker grades one request against every rubric in
// sequence — fine at tonight's volume, and easy to fan out further later.
func (s *Sampler) StartWorkers(ctx context.Context, n int) {
	for i := 0; i < n; i++ {
		go s.runWorker(ctx)
	}
}

func (s *Sampler) runWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case reqID := <-s.queue:
			s.gradeOne(ctx, reqID)
		}
	}
}

func (s *Sampler) gradeOne(ctx context.Context, requestID string) {
	ctx, span := s.otel.Tracer.Start(ctx, "eval.grade_request")
	defer span.End()
	span.SetAttributes(otelx.AttrVerigateRequestID.String(requestID))

	req, err := s.store.GetRequest(ctx, requestID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to load request")
		log.Printf("eval worker: failed to load request %s: %v", requestID, err)
		return
	}

	// A pure tool-call turn (tool_calls present, no text content) has
	// nothing meaningful for the text rubrics to grade — running
	// groundedness/format_compliance against an empty string would just
	// score near 0 for every legitimate tool call and quietly poison the
	// regression baseline. Text rubrics run whenever there's real text;
	// the tool-call rubric runs whenever there's a tool call. A response
	// with both (some models emit commentary alongside a tool call) gets
	// graded both ways, which is correct — both parts of the turn matter.
	if req.Response != "" {
		for _, rubric := range Rubrics {
			s.scoreAndRecord(ctx, span, req, rubric, req.Prompt, req.Response)
		}
	}
	if req.ToolCalls != "" {
		s.scoreAndRecord(ctx, span, req, ToolCallRubric, req.Prompt, req.ToolCalls)
	}
}

func (s *Sampler) scoreAndRecord(ctx context.Context, span trace.Span, req *store.RequestRecord, rubric Rubric, prompt, subject string) {
	v, err := s.judge.Score(ctx, rubric, prompt, subject)
	if err != nil {
		log.Printf("eval worker: judge call failed for %s/%s: %v", req.ID, rubric.Name, err)
		return
	}
	err = s.store.InsertEval(ctx, store.EvalRecord{
		RequestID: req.ID,
		Rubric:    rubric.Name,
		Score:     v.Score,
		Reasoning: v.Reasoning,
	})
	if err != nil {
		log.Printf("eval worker: failed to save eval for %s/%s: %v", req.ID, rubric.Name, err)
		return
	}

	span.AddEvent("eval.scored", trace.WithAttributes(
		otelx.AttrVerigateEvalRubric.String(rubric.Name),
		otelx.AttrVerigateEvalScore.Float64(v.Score),
	))
	s.otel.EvalScoreHist.Record(ctx, v.Score, metric.WithAttributes(
		attribute.String("verigate.eval.rubric", rubric.Name),
		otelx.AttrGenAIRequestModel.String(req.Model),
	))
}
