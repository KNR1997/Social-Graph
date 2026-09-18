// Package logger builds the process-wide structured logger on log/slog.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/kethaka-creskit/go-ddd-service/config"
)

// contextKey is the private key type for values this package injects.
type contextKey struct{ name string }

var requestIDKey = contextKey{name: "request_id"}

// New builds a slog logger from config. Text format for humans locally, JSON
// everywhere else.
func New(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.Log.Level)}

	var handler slog.Handler
	if strings.EqualFold(cfg.Log.Format, "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(&contextHandler{Handler: handler}).With(
		slog.String("service", cfg.App.Name),
		slog.String("env", cfg.App.Env),
	)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID returns a context carrying a request id for the logger to pick
// up. This is why every log call in the codebase uses the ...Context variants.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID reads the request id back out, if there is one.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// contextHandler copies request-scoped fields off the context onto every record,
// so handlers never have to pass the request id down by hand.
type contextHandler struct {
	slog.Handler
}

//nolint:gocritic // the slog.Handler interface fixes this signature; it cannot take a pointer.
func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}

	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}
