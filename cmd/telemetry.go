package cmd

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const instrumentationName = "github.com/Alexander-2212/kbot"

// tracer delegates to whatever provider initTelemetry installs globally.
var tracer = otel.Tracer(instrumentationName)

// instruments holds the bot metrics. In Prometheus they appear as
// kbot_commands_total and kbot_command_duration_seconds_*.
var instruments struct {
	commands metric.Int64Counter
	duration metric.Float64Histogram
}

// initTelemetry exports traces and metrics over OTLP/gRPC to the endpoint in
// the standard OTEL_EXPORTER_OTLP_ENDPOINT variable (OTEL_SERVICE_NAME,
// OTEL_RESOURCE_ATTRIBUTES and OTEL_METRIC_EXPORT_INTERVAL are honoured too).
// Without an endpoint the global no-op providers stay in place, so the bot
// still runs locally with no collector.
func initTelemetry(ctx context.Context) (shutdown func(context.Context) error, err error) {
	shutdown = func(context.Context) error { return nil }

	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		slog.Info("OTEL_EXPORTER_OTLP_ENDPOINT is not set, telemetry export disabled")
		return shutdown, createInstruments()
	}

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		slog.Warn("opentelemetry", "error", err)
	}))

	// Later options win: environment attributes override the defaults.
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", "kbot"),
			attribute.String("service.version", appVersion),
		),
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		if res == nil {
			return nil, err
		}
		slog.Warn("partial telemetry resource", "error", err)
	}

	// gRPC connects lazily: a collector that is not up yet is not fatal.
	traceExporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	metricExporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, errors.Join(err, tp.Shutdown(ctx))
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	shutdown = func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
	}
	return shutdown, createInstruments()
}

func createInstruments() error {
	meter := otel.Meter(instrumentationName)

	commands, err := meter.Int64Counter("kbot.commands",
		metric.WithDescription("Telegram messages handled, by command and status"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return err
	}

	duration, err := meter.Float64Histogram("kbot.command.duration",
		metric.WithDescription("Time to handle a message, including the reply to Telegram"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.05, 0.1, 0.25, 0.5, 1, 2.5, 5),
	)
	if err != nil {
		return err
	}

	instruments.commands = commands
	instruments.duration = duration
	return nil
}
