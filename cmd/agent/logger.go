package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

const (
	resetColor  = "\033[0m"
	greenColor  = "\033[32m"
	yellowColor = "\033[33m"
)

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getLogger(level string) *slog.Logger {
	handler := tint.NewTextHandler(os.Stdout, &tint.Options{
		Level:      parseLogLevel(level),
		TimeFormat: time.RFC3339,
	})

	return slog.New(handler)
}

func formatLogMsg(msg string) string {
	return fmt.Sprintf("%s%s%s", yellowColor, msg, resetColor)
}
