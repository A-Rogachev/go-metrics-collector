package main

import (
	"fmt"
	"time"

	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	handlers "github.com/A-Rogachev/go-metrics-collector/internal/handler"
	"github.com/A-Rogachev/go-metrics-collector/internal/middleware"
	services "github.com/A-Rogachev/go-metrics-collector/internal/service"
	"github.com/A-Rogachev/go-metrics-collector/internal/storage"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

func main() {

	cfg := config.GetServerConfig()
	logger := getLogger(cfg.LogLevel)
	appLogger := middleware.ZerologAdapter{Logger: logger}

	s, fs := storage.NewMemStorage(), storage.NewFileStorage(cfg.FileStoragePath)
	saver := services.NewMetricSaver(s, fs, time.Duration(cfg.StoreInterval), appLogger)

	if cfg.Restore {
		err := saver.RestoreMetrics()
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("Snapshot restored with errors: %s", err))
		}
	}
	go saver.PeriodicalSave()

	e := echo.New()
	e.Validator = &handlers.MetricValidator{Validator: validator.New()}

	e.Use(middleware.GetLoggingMiddleware(appLogger))
	e.Use(middleware.GzipRequestMiddleware(appLogger))
	e.Use(middleware.GzipResponseMiddleware)

	e.POST("update/", func(c *echo.Context) error {
		return handlers.UpdateMetricJSON(c, s, saver, appLogger)
	})
	e.POST("value/", func(c *echo.Context) error {
		return handlers.GetMetricValueJSON(c, s)
	})
	e.POST("update/:metricType/:key/:value", func(c *echo.Context) error {
		return handlers.UpdateMetric(c, s, saver, appLogger)
	})
	e.GET("value/:metricType/:key", func(c *echo.Context) error {
		return handlers.GetMetricValue(c, s)
	})
	e.GET("/", func(c *echo.Context) error {
		return handlers.GetAllMetrics(c, s)
	})

	if err := e.Start(cfg.ServerAddress); err != nil {
		logger.Fatal().Msg(fmt.Sprintf("failed to start server: %s", err))
	}

}
