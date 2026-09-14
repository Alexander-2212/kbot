package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	telebot "gopkg.in/telebot.v4"
)

var TeleToken = os.Getenv("TELE_TOKEN")

// startedAt is when this process started, reported by /uptime. After a
// rollout it shows how long the new pod has been serving.
var startedAt = time.Now()

var startCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"kbot"},
	Short:   "Start the telegram bot",
	Run: func(cmd *cobra.Command, args []string) {
		slog.SetDefault(newLogger(os.Stdout))

		if TeleToken == "" {
			fatal("TELE_TOKEN is empty: create a bot via @BotFather and export the token")
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		shutdownTelemetry, err := initTelemetry(ctx)
		if err != nil {
			fatal("cannot init telemetry", "error", err)
		}
		defer func() {
			// ctx is already cancelled here: flush the last spans and metrics
			// with a fresh deadline.
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdownTelemetry(flushCtx); err != nil {
				slog.Error("telemetry shutdown", "error", err)
			}
		}()

		kbot, err := telebot.NewBot(telebot.Settings{
			Token:  TeleToken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})
		if err != nil {
			fatal("cannot create bot", "error", err)
		}

		slog.Info("bot authorized", "username", kbot.Me.Username, "version", appVersion)

		kbot.Handle(telebot.OnText, textHandler)

		go kbot.Start()
		<-ctx.Done()
		slog.Info("shutting down")
		kbot.Stop()
	},
}

// textHandler serves one message as a trace: the root span covers the whole
// update, a child span covers the reply to the Telegram API. Every log line
// carries the trace_id, and the metrics count messages per command.
func textHandler(m telebot.Context) error {
	start := time.Now()
	payload := strings.ToLower(strings.TrimSpace(m.Text()))
	command := commandName(payload)

	var username, firstName string
	if sender := m.Sender(); sender != nil {
		username, firstName = sender.Username, sender.FirstName
	}

	ctx, span := tracer.Start(context.Background(), "kbot.command "+command,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("kbot.command", command),
			attribute.Int("telegram.update_id", m.Update().ID),
		),
	)
	defer span.End()

	slog.InfoContext(ctx, "message received", "from", username, "text", payload, "command", command)

	err := send(ctx, m, reply(ctx, command, payload, firstName))

	status := "ok"
	if err != nil {
		status = "error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "reply failed", "command", command, "error", err)
	}

	attrs := metric.WithAttributes(
		attribute.String("command", command),
		attribute.String("status", status),
	)
	instruments.commands.Add(ctx, 1, attrs)
	instruments.duration.Record(ctx, time.Since(start).Seconds(), attrs)
	return err
}

func send(ctx context.Context, m telebot.Context, text string) error {
	ctx, span := tracer.Start(ctx, "telegram sendMessage", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	start := time.Now()
	err := m.Send(text)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	slog.InfoContext(ctx, "reply sent", "duration_ms", time.Since(start).Milliseconds())
	return nil
}

// commandName maps a message to a small fixed set of names, which keeps metric
// label cardinality bounded whatever users type.
func commandName(payload string) string {
	// "/help@kbot_bot" is how commands arrive in group chats.
	name, _, _ := strings.Cut(strings.TrimPrefix(payload, "/"), "@")
	switch name {
	case "start", "hello":
		return "hello"
	case "version", "ping", "time", "host", "uptime", "trace", "help":
		return name
	}
	return "unknown"
}

func reply(ctx context.Context, command, payload, firstName string) string {
	switch command {
	case "hello":
		return fmt.Sprintf("Hello, %s! I'm kbot %s", firstName, appVersion)
	case "version":
		return appVersion
	case "ping":
		return "pong"
	case "time":
		return time.Now().Format(time.RFC1123)
	case "host":
		// In Kubernetes HOSTNAME is the pod name, so this shows which pod (and,
		// after a rollout, which ReplicaSet) is serving the bot.
		host, err := os.Hostname()
		if err != nil {
			host = "unknown"
		}
		return fmt.Sprintf("%s, version %s", host, appVersion)
	case "uptime":
		return fmt.Sprintf("up %s, version %s", time.Since(startedAt).Round(time.Second), appVersion)
	case "trace":
		// The same ID is in this request's log lines and in Tempo.
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			return "trace_id " + sc.TraceID().String()
		}
		return "tracing is disabled"
	case "help":
		return "Available commands:\nhello — greeting\nping — pong\nversion — bot version\ntime — current server time\nuptime — time since start and version\nhost — pod serving this bot\ntrace — trace ID of this request\nhelp — this message"
	default:
		return fmt.Sprintf("Unknown command: %q\nTry /help", payload)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
}
