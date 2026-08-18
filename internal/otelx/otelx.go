// Package otelx wires up OpenTelemetry for Verigate. By default (no
// OTEL_EXPORTER_OTLP_ENDPOINT set) it exports spans and metrics to stdout —
// so instrumentation correctness is provable with zero external
// infrastructure. Set OTEL_EXPORTER_OTLP_ENDPOINT to point the exact same
// instrumentation at a real OTel Collector, Grafana, Honeycomb, Datadog, or
// anything else that speaks OTLP — that swap is the whole point of using
// OpenTelemetry instead of a bespoke logging format.
package otelx

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/rakshit-gen/verigate"

type Providers struct {
	Tracer   trace.Tracer
	Shutdown func(context.Context) error

	RequestCounter metric.Int64Counter
	LatencyHist    metric.Float64Histogram
	TokenCounter   metric.Int64Counter
	EvalScoreHist  metric.Float64Histogram
	TTFTHist       metric.Float64Histogram
}

// Init sets up global tracer/meter providers and returns the instruments the
// rest of the app records against. Exporter choice is env-driven, matching
// standard OTel SDK conventions (OTEL_EXPORTER_OTLP_ENDPOINT), so this is
// unsurprising to anyone who's wired OTel into a service before.
func Init(ctx context.Context, serviceName string) (*Providers, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	tp, tracerShutdown, err := setupTracing(ctx, res, endpoint)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(tp)

	mp, meterShutdown, err := setupMetrics(ctx, res, endpoint)
	if err != nil {
		return nil, err
	}
	otel.SetMeterProvider(mp)

	tracer := tp.Tracer(instrumentationName)
	meter := mp.Meter(instrumentationName)

	requestCounter, err := meter.Int64Counter(
		"gen_ai.client.request.count",
		metric.WithDescription("Number of chat completion requests handled by the gateway"),
	)
	if err != nil {
		return nil, err
	}
	latencyHist, err := meter.Float64Histogram(
		"gen_ai.client.operation.duration",
		metric.WithDescription("End-to-end latency of a chat completion request"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}
	tokenCounter, err := meter.Int64Counter(
		"gen_ai.client.token.usage",
		metric.WithDescription("Tokens consumed per request, split by input/output via attribute"),
	)
	if err != nil {
		return nil, err
	}
	// There is no official GenAI semantic-convention metric for continuous
	// quality evaluation yet (the spec covers request/response/token shape,
	// not post-hoc grading) — this is Verigate's own extension, namespaced
	// accordingly rather than squatting on the gen_ai.* prefix.
	evalScoreHist, err := meter.Float64Histogram(
		"verigate.eval.score",
		metric.WithDescription("LLM-judge score for a sampled response, 0-1, by rubric"),
	)
	if err != nil {
		return nil, err
	}
	// gen_ai.server.time_to_first_token is a real GenAI semantic-convention
	// metric name — the one place tonight's instrumentation uses the spec's
	// own naming for something Verigate itself computes, since TTFT only
	// exists once you're actually proxying a stream rather than a single
	// buffered response.
	ttftHist, err := meter.Float64Histogram(
		"gen_ai.server.time_to_first_token",
		metric.WithDescription("Time from request start to the first streamed content chunk"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &Providers{
		Tracer: tracer,
		Shutdown: func(ctx context.Context) error {
			if err := tracerShutdown(ctx); err != nil {
				return err
			}
			return meterShutdown(ctx)
		},
		RequestCounter: requestCounter,
		LatencyHist:    latencyHist,
		TokenCounter:   tokenCounter,
		EvalScoreHist:  evalScoreHist,
		TTFTHist:       ttftHist,
	}, nil
}

func setupTracing(ctx context.Context, res *resource.Resource, endpoint string) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	var exporter sdktrace.SpanExporter
	var err error

	if endpoint != "" {
		exporter, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	return tp, tp.Shutdown, nil
}

func setupMetrics(ctx context.Context, res *resource.Resource, endpoint string) (*sdkmetric.MeterProvider, func(context.Context) error, error) {
	var exporter sdkmetric.Exporter
	var err error

	if endpoint != "" {
		exporter, err = otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(endpoint))
	} else {
		exporter, err = stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	}
	if err != nil {
		return nil, nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)
	return mp, mp.Shutdown, nil
}
