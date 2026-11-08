package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/A-Rogachev/go-metrics-collector/internal/config"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

const BatchSize = 50

type AgentStorage struct {
	mutex    sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func (storage *AgentStorage) snapshot() models.MetricSnapshot {
	storage.mutex.Lock()
	gauges, counters := maps.Clone(storage.gauges), maps.Clone(storage.counters)
	storage.mutex.Unlock()
	return models.MetricSnapshot{Gauges: gauges, Counters: counters}
}

func (storage *AgentStorage) subtractCounters(sentCounters map[string]int64) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	for key, value := range sentCounters {
		storage.counters[key] -= value
		if storage.counters[key] < 0 {
			storage.counters[key] = 0
		}
	}
}

func (storage *AgentStorage) collect(polling bool, interval int) {
	for {
		m := runtime.MemStats{}
		runtime.ReadMemStats(&m)
		storage.mutex.Lock()
		storage.gauges = map[string]float64{
			"Alloc":         float64(m.Alloc),
			"BuckHashSys":   float64(m.BuckHashSys),
			"Frees":         float64(m.Frees),
			"GCCPUFraction": m.GCCPUFraction,
			"GCSys":         float64(m.GCSys),
			"HeapAlloc":     float64(m.HeapAlloc),
			"HeapIdle":      float64(m.HeapIdle),
			"HeapInuse":     float64(m.HeapInuse),
			"HeapObjects":   float64(m.HeapObjects),
			"HeapReleased":  float64(m.HeapReleased),
			"HeapSys":       float64(m.HeapSys),
			"LastGC":        float64(m.LastGC),
			"Lookups":       float64(m.Lookups),
			"MCacheInuse":   float64(m.MCacheInuse),
			"MCacheSys":     float64(m.MCacheSys),
			"MSpanInuse":    float64(m.MSpanInuse),
			"MSpanSys":      float64(m.MSpanSys),
			"Mallocs":       float64(m.Mallocs),
			"NextGC":        float64(m.NextGC),
			"NumForcedGC":   float64(m.NumForcedGC),
			"NumGC":         float64(m.NumGC),
			"OtherSys":      float64(m.OtherSys),
			"PauseTotalNs":  float64(m.PauseTotalNs),
			"StackInuse":    float64(m.StackInuse),
			"StackSys":      float64(m.StackSys),
			"Sys":           float64(m.Sys),
			"TotalAlloc":    float64(m.TotalAlloc),
			"RandomValue":   rand.Float64(),
		}
		storage.counters["PollCount"]++
		storage.mutex.Unlock()
		if !polling {
			return
		}
		time.Sleep(time.Second * time.Duration(interval))
	}
}

func sendRequests(client *http.Client, logger *slog.Logger, snapshot models.MetricSnapshot, baseURL string) map[string]int64 {
	sentCounters := make(map[string]int64, 0)
	for key, value := range snapshot.Gauges {
		url := fmt.Sprintf("%s/gauge/%s/%f", baseURL, key, value)
		if err := sendMetric(client, url); err != nil {
			logger.Debug(err.Error())
		}
	}
	for key, value := range snapshot.Counters {
		url := fmt.Sprintf("%s/counter/%s/%d", baseURL, key, value)
		if err := sendMetric(client, url); err != nil {
			logger.Debug(err.Error())
			continue
		}
		sentCounters[key] = value
	}
	logger.Info("Metrics sent to the server")
	return sentCounters
}

func sendMetric(client *http.Client, url string) error {
	response, err := client.Post(url, "text/plain", nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", response.StatusCode)
	}
	_, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		return err
	}
	return nil
}

func sendJSONBatchRequests(
	client *http.Client,
	logger *slog.Logger,
	snapshot models.MetricSnapshot,
	baseURL string,
) map[string]int64 {
	sentCounters := make(map[string]int64, 0)
	var metricsToSend []models.Metric
	for key, value := range snapshot.Gauges {
		metricsToSend = append(
			metricsToSend,
			models.Metric{
				ID:    key,
				MType: models.Gauge,
				Value: &value,
			},
		)
	}
	for key, value := range snapshot.Counters {
		metricsToSend = append(
			metricsToSend,
			models.Metric{
				ID:    key,
				MType: models.Counter,
				Delta: &value,
			},
		)
		sentCounters[key] = value
	}
	if len(metricsToSend) > 0 {
		err := sendJSONMetricBatches(client, baseURL+"/", metricsToSend)
		if err != nil {
			logger.Debug(err.Error())
			return make(map[string]int64, 0)
		} else {
			logger.Info("Metrics sent to the server")
			return sentCounters
		}
	}
	logger.Info("No metrics to send")
	return sentCounters
}

func sendJSONMetricBatches(client *http.Client, url string, metrics []models.Metric) error {
	for start := 0; start < len(metrics); start += BatchSize {
		end := start + BatchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		if err := sendJSONMetricBatch(
			client,
			url,
			metrics[start:end],
		); err != nil {
			return err
		}
	}
	return nil
}

func sendJSONMetricBatch(client *http.Client, url string, metrics []models.Metric) error {
	jsonBody, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	var compressedBody bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBody)
	if _, err := gzipWriter.Write(jsonBody); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, url, &compressedBody)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", response.StatusCode)
	}
	_, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		return err
	}
	return nil
}

func sendJSONRequests(
	client *http.Client,
	logger *slog.Logger,
	snapshot models.MetricSnapshot,
	baseURL string,
) map[string]int64 {
	sentCounters := make(map[string]int64, 0)
	url := baseURL + "/"
	for key, value := range snapshot.Gauges {
		reqBody := models.Metric{
			ID:    key,
			MType: models.Gauge,
			Value: &value,
		}
		if err := sendJSONMetric(client, url, reqBody); err != nil {
			logger.Debug(err.Error())
		}
	}
	for key, value := range snapshot.Counters {
		reqBody := models.Metric{
			ID:    key,
			MType: models.Counter,
			Delta: &value,
		}
		if err := sendJSONMetric(client, url, reqBody); err != nil {
			logger.Debug(err.Error())
			continue
		}
		sentCounters[key] = value
	}
	logger.Info("Metrics sent to the server")
	return sentCounters
}

func sendJSONMetric(client *http.Client, url string, body models.Metric) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var compressedBody bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBody)
	if _, err := gzipWriter.Write(jsonBody); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, url, &compressedBody)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", response.StatusCode)
	}
	_, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	cfg := config.GetAgentConfig()
	logger := getLogger(cfg.Additional.LogLevel)

	tempStorage := AgentStorage{
		gauges:   make(map[string]float64),
		counters: map[string]int64{"PollCount": 0},
	}

	go tempStorage.collect(true, cfg.PollConfig.PollInterval)

	client := http.Client{}
	for {
		time.Sleep(time.Second * time.Duration(cfg.ReportConfig.ReportInterval))

		snapshot := tempStorage.snapshot()

		sentCounters := sendJSONBatchRequests(&client, logger, snapshot, "http://"+cfg.APIConfig.Address+"/updates")
		// NOTE: старая реализация с отправкой метрик по одному в json
		// sentCounters := sendJSONRequests(&client, logger, snapshot, "http://"+cfg.APIConfig.Address+"/update")
		// NOTE: старая реализация с отправкой метрик через параметры запроса
		//sentCounters := sendRequests(&client, logger, snapshot, "http://"+cfg.APIConfig.Address+"/update")
		tempStorage.subtractCounters(sentCounters)
	}
}
