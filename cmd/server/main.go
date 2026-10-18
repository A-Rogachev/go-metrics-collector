package main

import (
	"fmt"

	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	handlers "github.com/A-Rogachev/go-metrics-collector/internal/handler"
	internal_middleware "github.com/A-Rogachev/go-metrics-collector/internal/middleware"
	mem_storage "github.com/A-Rogachev/go-metrics-collector/internal/storage"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

func main() {

	cfg := config.GetServerConfig()
	storage := mem_storage.NewMemStorage()
	logger := getLogger(cfg.LogLevel)
	appLogger := internal_middleware.ZerologAdapter{Logger: logger}

	e := echo.New()
	e.Validator = &handlers.MetricValidator{Validator: validator.New()}

	e.Use(internal_middleware.GetLoggingMiddleware(appLogger))
	e.Use(internal_middleware.GzipRequestMiddleware)
	e.Use(internal_middleware.GzipResponseMiddleware)

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
