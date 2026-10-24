package main

import (
	"fmt"

	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	handlers "github.com/A-Rogachev/go-metrics-collector/internal/handler"
	"github.com/A-Rogachev/go-metrics-collector/internal/middleware"
	"github.com/A-Rogachev/go-metrics-collector/internal/storage"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

func main() {

	cfg := config.GetServerConfig()
	logger := getLogger(cfg.LogLevel)
	appLogger := middleware.ZerologAdapter{Logger: logger}

	s := storage.NewMemStorage()
	fs := storage.NewFileStorage(cfg.FileStoragePath, cfg.StoreInterval)
	if cfg.Restore {
		snapshot, err := fs.LoadMetricsFromFile()
		if err != nil {
			logger.Error().Msg("Snapshot restored with errors")
		}
		loadingError := s.LoadFromSnapshot(snapshot)
		if loadingError != nil {
			logger.Error().Msg("Can't restore metrics from file.")
		}
	}
	go fs.PeriodicSaveToFile(s)

	e := echo.New()
	e.Validator = &handlers.MetricValidator{Validator: validator.New()}

	e.Use(middleware.GetLoggingMiddleware(appLogger))
	e.Use(middleware.GzipRequestMiddleware)
	e.Use(middleware.GzipResponseMiddleware)

	e.POST("update/", func(c *echo.Context) error {
		return handlers.UpdateMetricJSON(c, s, fs, appLogger)
	})
	e.POST("value/", func(c *echo.Context) error {
		return handlers.GetMetricValueJSON(c, s)
	})
	e.POST("update/:metricType/:key/:value", func(c *echo.Context) error {
		return handlers.UpdateMetric(c, s, fs, appLogger)
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
