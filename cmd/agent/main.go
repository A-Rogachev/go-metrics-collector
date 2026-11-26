package main

import (
	"crypto/sha256"
	"net/http"
	"sync"
	"time"

	"github.com/A-Rogachev/go-metrics-collector/internal/agent"
	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

type MetricSender interface {
	SendMetrics(models.MetricSnapshot, int) map[string]int64
	SendMetricsPeriodically(
		<-chan models.MetricSnapshot,
		chan<- map[string]int64,
		time.Duration,
		int,
	)
}

func main() {
	cfg := config.GetAgentConfig()
	logger := getLogger(cfg.Additional.LogLevel)

	tempStorage := agent.NewAgentStorage()

	periodicPolling := true
	pollInterval := time.Duration(cfg.PollConfig.PollInterval)
	tempStorage.Collect(periodicPolling, pollInterval)

	client := &http.Client{}
	url := "http://" + cfg.APIConfig.Address + "/updates"
	var sender MetricSender = agent.NewJSONBatchSender(
		client,
		logger,
		url,
		agent.NewMetricSigner(cfg.SecretKey, sha256.New),
	)

	shapshotChan := make(chan models.MetricSnapshot)
	countersChan := make(chan map[string]int64)

	var wg sync.WaitGroup
	reportInterval := time.Duration(cfg.ReportConfig.ReportInterval)
	wg.Go(func() { tempStorage.PeriodicalSnapshot(shapshotChan, reportInterval) })
	wg.Go(func() {
		sender.SendMetricsPeriodically(
			shapshotChan,
			countersChan,
			reportInterval,
			cfg.RateLimit,
		)
	})
	wg.Go(func() { tempStorage.SubtractCounters(countersChan) })
	wg.Wait()

}
