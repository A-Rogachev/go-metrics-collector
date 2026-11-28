package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/A-Rogachev/go-metrics-collector/internal/agent"
	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
	"github.com/A-Rogachev/go-metrics-collector/internal/signing"
)

type MetricSender interface {
	SendMetrics(models.MetricSnapshot, int) map[string]int64
	SendMetricsPeriodically(
		context.Context,
		<-chan models.MetricSnapshot,
		chan<- map[string]int64,
		int,
		*slog.Logger,
	)
}

func main() {
	cfg := config.GetAgentConfig()
	logger := getLogger(cfg.Additional.LogLevel)

	tempStorage := agent.NewAgentStorage()

	periodicPolling := true
	pollInterval := time.Duration(cfg.PollConfig.PollInterval) * time.Second

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	tempStorage.Collect(ctx, periodicPolling, pollInterval, logger)

	client := &http.Client{}
	url := "http://" + cfg.APIConfig.Address + "/updates"
	var sender MetricSender = agent.NewJSONBatchSender(
		client,
		logger,
		url,
		signing.NewHMACSigner(
			cfg.SecretKey,
			sha256.New,
			base64.StdEncoding,
		),
	)

	shapshotChan := make(chan models.MetricSnapshot)
	countersChan := make(chan map[string]int64)

	var wg sync.WaitGroup
	reportInterval := time.Duration(cfg.ReportConfig.ReportInterval) * time.Second

	wg.Go(func() { tempStorage.PeriodicalSnapshot(ctx, shapshotChan, reportInterval, logger) })
	wg.Go(func() {
		sender.SendMetricsPeriodically(
			ctx,
			shapshotChan,
			countersChan,
			cfg.RateLimit,
			logger,
		)
	})
	wg.Go(func() { tempStorage.SubtractCounters(ctx, countersChan, logger) })
	wg.Wait()
	logger.Info("- - - Agent stopped gracefully")
}
