package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	yellowColor string = "\033[33m"
	redColor    string = "\033[31m"
	greenColor  string = "\033[32m"
	resetColor  string = "\033[0m"
)

func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func getLogger(level string) *zerolog.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatTimestamp = func(value any) string {
		return yellowColor + fmt.Sprint(value) + resetColor
	}
	output.FormatLevel = func(i any) string {
		levelName := fmt.Sprint(i)
		var color string
		switch levelName {
		case "info":
			color = greenColor
		case "warn":
			color = yellowColor
		case "error", "fatal", "panic":
			color = redColor
		default:
			color = resetColor
		}
		return color + strings.ToUpper(fmt.Sprintf("| %-6s|", levelName)) + resetColor
	}
	logger := zerolog.New(output).Level(parseLogLevel(level)).With().Timestamp().Logger()
	return &logger
}
