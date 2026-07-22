// Package logger centralises structured JSON logging via log/slog so every
// entrypoint and module logs with the same format and field conventions.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON slog.Logger at the given level ("debug"/"info"/"warn"/"error").
func New(level string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
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
