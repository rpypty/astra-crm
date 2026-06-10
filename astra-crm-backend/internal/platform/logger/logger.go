package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(appEnv string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if strings.EqualFold(appEnv, "development") || strings.EqualFold(appEnv, "test") {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
