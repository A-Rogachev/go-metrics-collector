package main

import (
	"fmt"
	"time"

	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	handlers "github.com/A-Rogachev/go-metrics-collector/internal/handler"
	"github.com/A-Rogachev/go-metrics-collector/internal/middleware"
	services "github.com/A-Rogachev/go-metrics-collector/internal/service"
	"github.com/A-Rogachev/go-metrics-collector/internal/storage"
	"github.com/A-Rogachev/go-metrics-collector/migrations"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

func main() {

	cfg, cfgDB := config.GetServerConfig(), config.GetDatabaseConfig()

	logger := getLogger(cfg.LogLevel)
	appLogger := middleware.ZerologAdapter{Logger: logger}

	dbStorage, err := storage.NewDBStorage(cfgDB.DriverName, cfgDB.DSN)
	if err != nil {
		logger.Fatal().Msg(fmt.Sprintf("database error: %s", err))
	}
	if err := migrations.Run(dbStorage.GetPool()); err != nil {
		logger.Fatal().Msg(fmt.Sprintf("database migrations error: %s", err))
	}

	ms, fs := storage.NewMemStorage(), storage.NewFileStorage(cfg.FileStoragePath)
	saver := services.NewMetricSaver(
		ms,
		fs,
		time.Duration(cfg.StoreInterval)*time.Second,
		appLogger,
	)

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
		return handlers.UpdateMetricJSON(c, ms, saver, appLogger)
	})
	e.POST("value/", func(c *echo.Context) error {
		return handlers.GetMetricValueJSON(c, ms)
	})
	e.POST("update/:metricType/:key/:value", func(c *echo.Context) error {
		return handlers.UpdateMetric(c, ms, saver, appLogger)
	})
	e.GET("value/:metricType/:key", func(c *echo.Context) error {
		return handlers.GetMetricValue(c, ms)
	})
	e.GET("/", func(c *echo.Context) error {
		return handlers.GetAllMetrics(c, ms)
	})
	e.GET("/ping", func(c *echo.Context) error {
		return handlers.PingDatabase(c, dbStorage)
	})

	if err := e.Start(cfg.ServerAddress); err != nil {
		logger.Fatal().Msg(fmt.Sprintf("failed to start server: %s", err))
	}

}
