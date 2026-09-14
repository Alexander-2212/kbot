package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestCommandName(t *testing.T) {
	cases := map[string]string{
		"/start":         "hello",
		"hello":          "hello",
		"/help":          "help",
		"/help@kbot_bot": "help",
		"uptime":         "uptime",
		"/trace":         "trace",
		"what is this":   "unknown",
		"":               "unknown",
	}
	for payload, want := range cases {
		if got := commandName(payload); got != want {
			t.Errorf("commandName(%q) = %q, want %q", payload, got, want)
		}
	}
}

func TestLoggerAddsTraceID(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	var buf bytes.Buffer
	newLogger(&buf).InfoContext(ctx, "message received", "command", "ping")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log line is not JSON: %v: %s", err, buf.String())
	}
	if got, want := line["trace_id"], span.SpanContext().TraceID().String(); got != want {
		t.Errorf("trace_id = %v, want %v", got, want)
	}
	if got, want := line["span_id"], span.SpanContext().SpanID().String(); got != want {
		t.Errorf("span_id = %v, want %v", got, want)
	}
}

func TestLoggerWithoutSpan(t *testing.T) {
	var buf bytes.Buffer
	newLogger(&buf).InfoContext(context.Background(), "no span")
	if strings.Contains(buf.String(), "trace_id") {
		t.Errorf("unexpected trace_id without an active span: %s", buf.String())
	}
}
