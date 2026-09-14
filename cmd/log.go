package cmd

import (
	"context"
	"io"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// traceHandler adds trace_id and span_id of the active span to every record
// logged with a context, so a log line in Loki leads to its trace in Tempo
// and a trace leads back to its log lines.
type traceHandler struct {
	slog.Handler
}

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{h.Handler.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{h.Handler.WithGroup(name)}
}

// newLogger writes JSON lines that Fluent Bit parses into separate fields.
func newLogger(w io.Writer) *slog.Logger {
	return slog.New(traceHandler{slog.NewJSONHandler(w, nil)})
}

// fatal logs the error and exits, like log.Fatal did.
func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
