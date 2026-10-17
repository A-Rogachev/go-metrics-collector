package main

import (
	"fmt"

	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	handlers "github.com/A-Rogachev/go-metrics-collector/internal/handler"
	mem_storage "github.com/A-Rogachev/go-metrics-collector/internal/storage"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {

	cfg := config.GetServerConfig()
	storage := mem_storage.NewMemStorage()
	logger := getLogger(cfg.LogLevel)

	e := echo.New()
	e.Validator = &handlers.MetricValidator{Validator: validator.New()}
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
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
			if v.Error != nil {
				logger.Error().Msg(message)
			} else {
				logger.Info().Msg(message)
			}
			return nil
		},
	}))
	e.POST("update/", func(c *echo.Context) error {
		return handlers.UpdateMetricJSON(c, storage)
	})
	e.POST("value/", func(c *echo.Context) error {
		return handlers.GetMetricValueJSON(c, storage)
	})
	e.POST("update/:metricType/:key/:value", func(c *echo.Context) error {
		return handlers.UpdateMetric(c, storage)
	})
	e.GET("value/:metricType/:key", func(c *echo.Context) error {
		return handlers.GetMetricValue(c, storage)
	})
	e.GET("/", func(c *echo.Context) error {
		return handlers.GetAllMetrics(c, storage)
	})

	if err := e.Start(cfg.ServerAddress); err != nil {
		logger.Fatal().Msg(fmt.Sprintf("failed to start server: %s", err))
	}
}
