package main

import (
	"net/http"

	"time"

	"github.com/A-Rogachev/go-metrics-collector/internal/agent"
	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

type MetricSender interface {
	SendMetrics(models.MetricSnapshot) map[string]int64
}

func main() {
	cfg := config.GetAgentConfig()
	logger := getLogger(cfg.Additional.LogLevel)

	tempStorage := agent.NewAgentStorage()
	go tempStorage.Collect(true, cfg.PollConfig.PollInterval)

	var sender MetricSender = agent.NewJSONBatchSender(
		&http.Client{},
		logger,
		"http://"+cfg.APIConfig.Address+"/updates",
	)
	// NOTE: старая реализация с отправкой метрик по одному в json
	// var sender MetricSender = agent.NewJSONSender(&http.Client{}, logger, "http://"+cfg.APIConfig.Address+"/update")
	// NOTE: старая реализация с отправкой метрик через параметры запроса
	// sender = agent.NewQueryParamsSender(&http.Client{}, logger, "http://"+cfg.APIConfig.Address+"/update")
	for {
		time.Sleep(time.Second * time.Duration(cfg.ReportConfig.ReportInterval))
		snapshot := tempStorage.Snapshot()
		sentCounters := sender.SendMetrics(snapshot)
		tempStorage.SubtractCounters(sentCounters)
	}
}
