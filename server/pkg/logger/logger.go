package logger

import (
	"context"
	"log/slog"
	"os"
)

var programLevel = new(slog.LevelVar)

func init() {
	programLevel.Set(slog.LevelInfo)
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))
}

func SetLevel(level slog.Level) {
	programLevel.Set(level)
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

func WithContext(ctx context.Context, args ...any) *slog.Logger {
	return slog.With(args...)
}
