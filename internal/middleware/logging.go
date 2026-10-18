package middleware

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rs/zerolog"
)

type LoggerEvent interface {
	Msg(string)
}

type Logger interface {
	Error() LoggerEvent
	Info() LoggerEvent
}

type ZerologAdapter struct {
	Logger *zerolog.Logger
}

func (l ZerologAdapter) Error() LoggerEvent {
	return l.Logger.Error()
}

func (l ZerologAdapter) Info() LoggerEvent {
	return l.Logger.Info()
}

func GetLoggingMiddleware(logger Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:           true,
		LogMethod:        true,
		LogStatus:        true,
		LogContentLength: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			message := fmt.Sprintf(
				"%d %s %s (contentLength=%s)",
				v.Status,
				v.Method,
				v.URI,
				v.ContentLength,
			)
			if v.Status >= http.StatusBadRequest {
				logger.Error().Msg(message)
			} else {
				logger.Info().Msg(message)
			}
			return nil
		},
	})
}
