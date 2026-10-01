package main

import (
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
)

type metricStorage struct {
	mutex    sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

type metricSnapshot struct {
	gauges   map[string]float64
	counters map[string]int64
}

func (storage *metricStorage) snapshot() metricSnapshot {
	storage.mutex.Lock()
	gauges, counters := maps.Clone(storage.gauges), maps.Clone(storage.counters)
	storage.mutex.Unlock()
	return metricSnapshot{gauges, counters}
}

func (storage *metricStorage) subtractCounters(sentCounters map[string]int64) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	for key, value := range sentCounters {
		storage.counters[key] -= value
		if storage.counters[key] < 0 {
			storage.counters[key] = 0
		}
	}
}

func (storage *metricStorage) collect(polling bool, interval int) {
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

func SendRequests(client *http.Client, snapshot metricSnapshot, baseURL string) map[string]int64 {
	sentCounters := make(map[string]int64, 0)
	for key, value := range snapshot.gauges {
		url := fmt.Sprintf("%s/gauge/%s/%f", baseURL, key, value)
		if err := sendMetric(client, url); err != nil {
			slog.Debug(err.Error())
		}
	}
	for key, value := range snapshot.counters {
		url := fmt.Sprintf("%s/counter/%s/%d", baseURL, key, value)
		if err := sendMetric(client, url); err != nil {
			slog.Debug(err.Error())
			continue
		}
		sentCounters[key] = value
	}
	slog.Info("Metrics sent to the server")
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

func main() {
	cfg := config.GetAgentConfig()
	setupLogger(cfg.Additional.LogLevel)

	tempStorage := metricStorage{
		gauges:   make(map[string]float64),
		counters: map[string]int64{"PollCount": 0},
	}

	go tempStorage.collect(true, cfg.PollConfig.PollInterval)

	client := http.Client{}
	for {
		time.Sleep(time.Second * time.Duration(cfg.ReportConfig.ReportInterval))

		snapshot := tempStorage.snapshot()
		sentCounters := SendRequests(&client, snapshot, "http://"+cfg.APIConfig.Address+"/update")
		tempStorage.subtractCounters(sentCounters)
	}
}
