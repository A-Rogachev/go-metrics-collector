package main

import (
	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	handlers "github.com/A-Rogachev/go-metrics-collector/internal/handler"
	mem_storage "github.com/A-Rogachev/go-metrics-collector/internal/storage"
	"github.com/labstack/echo/v5"
)

func main() {

	cfg := config.GetServerConfig()

	e := echo.New()
	storage := mem_storage.NewMemStorage()

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
		e.Logger.Error("failed to start server", "error", err)
	}
}
