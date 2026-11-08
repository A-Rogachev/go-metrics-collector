package main

import (
	"fmt"
	"time"

	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	handlers "github.com/A-Rogachev/go-metrics-collector/internal/handler"
	"github.com/A-Rogachev/go-metrics-collector/internal/middleware"
	repositories "github.com/A-Rogachev/go-metrics-collector/internal/repository"
	services "github.com/A-Rogachev/go-metrics-collector/internal/service"
	"github.com/A-Rogachev/go-metrics-collector/internal/storage"
	"github.com/A-Rogachev/go-metrics-collector/migrations"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

func main() {

	cfg, cfgDB := config.GetServerConfig(), config.GetDatabaseConfig()
	log := middleware.ZerologAdapter{Logger: getLogger(cfg.LogLevel)}

	memoryStorage := storage.NewMemStorage()

	var saver handlers.MetricSaver
	var dbHealthChecker handlers.DBHealthChecker = storage.UnavailableDBHealthChecker{}

	switch {
	case cfgDB.DSN != "":
		dbStorage, err := storage.NewDBStorage(cfgDB.DriverName, cfgDB.DSN)
		if err != nil {
			log.Logger.Fatal().Msg(fmt.Sprintf("database error: %s", err))
		}
		if err := migrations.Run(dbStorage.GetPool()); err != nil {
			log.Logger.Fatal().Msg(fmt.Sprintf("database migrations error: %s", err))
		}
		repository := repositories.NewPostgresMetricsRepository(dbStorage.GetPool())
		saver = services.NewMetricDBSaver(repository)
		dbHealthChecker = dbStorage
	case cfg.FileStoragePath != "":
		fileStorage := storage.NewFileStorage(cfg.FileStoragePath)
		fileSaver := services.NewMetricSaver(
			memoryStorage,
			fileStorage,
			time.Duration(cfg.StoreInterval)*time.Second,
			log,
		)
		saver = fileSaver
		if cfg.Restore {
			err := fileSaver.RestoreMetrics()
			if err != nil {
				log.Logger.Error().Msg(fmt.Sprintf("Snapshot restored with errors: %s", err))
			}
		}
		go fileSaver.PeriodicalSave()
	default:
		saver = services.NewNoopMetricSaver()
	}

	e := echo.New()
	e.Validator = &handlers.MetricValidator{Validator: validator.New()}

	e.Use(middleware.GetLoggingMiddleware(log))
	e.Use(middleware.GzipRequestMiddleware(log))
	e.Use(middleware.GzipResponseMiddleware)

	e.POST("update/", func(c *echo.Context) error {
		return handlers.UpdateMetricJSON(c, memoryStorage, saver, log)
	})
	e.POST("value/", func(c *echo.Context) error {
		return handlers.GetMetricValueJSON(c, memoryStorage)
	})
	e.POST("update/:metricType/:key/:value", func(c *echo.Context) error {
		return handlers.UpdateMetric(c, memoryStorage, saver, log)
	})
	e.GET("value/:metricType/:key", func(c *echo.Context) error {
		return handlers.GetMetricValue(c, memoryStorage)
	})
	e.GET("/", func(c *echo.Context) error {
		return handlers.GetAllMetrics(c, memoryStorage)
	})
	e.GET("/ping", func(c *echo.Context) error {
		return handlers.PingDatabase(c, dbHealthChecker)
	})

	if err := e.Start(cfg.ServerAddress); err != nil {
		log.Logger.Fatal().Msg(fmt.Sprintf("failed to start server: %s", err))
	}

}
