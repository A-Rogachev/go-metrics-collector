package main

import (
	"fmt"
	"io"
	"log"
	"maps"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"
)

const (
	pollInterval   int    = 2
	reportInterval int    = 10
	baseServerUrl  string = "http://localhost:8080/update"
)

type metricStorage struct {
	mutex    sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

type metricSnapshot struct {
	gauges   map[string]float64
	counters map[string]int64
}

func (storage *metricStorage) snapshot() metricSnapshot {
	storage.mutex.RLock()
	gauges, counters := maps.Clone(storage.gauges), maps.Clone(storage.counters)
	storage.mutex.RUnlock()
	return metricSnapshot{gauges, counters}
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
		if polling == false {
			return
		}
		time.Sleep(time.Second * time.Duration(interval))
	}
}

func SendRequests(client *http.Client, snapshot metricSnapshot, baseUrl string) {
	gauges, counters := snapshot.gauges, snapshot.counters
	for name, value := range gauges {
		url := fmt.Sprintf("%s/gauge/%s/%f", baseUrl, name, value)
		_, err := sendMetric(client, url)
		if err != nil {
			log.Println(err)
		}
	}
	for name, value := range counters {
		url := fmt.Sprintf("%s/counter/%s/%d", baseUrl, name, value)
		_, err := sendMetric(client, url)
		if err != nil {
			log.Println(err)
		}
	}
	log.Println("metrics sent")
}

func sendMetric(client *http.Client, url string) (bool, error) {
	response, err := client.Post(url, "text/plain", nil)
	if err != nil {
		return false, err
	}
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status: %d", response.StatusCode)
	}
	defer response.Body.Close()
	_, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		return false, err
	}
	return true, nil
}

func main() {
	tempStorage := metricStorage{
		gauges:   make(map[string]float64),
		counters: map[string]int64{"PollCount": 0},
	}

	go tempStorage.collect(true, pollInterval)

	client := http.Client{}
	for {
		time.Sleep(time.Second * time.Duration(reportInterval))
		SendRequests(&client, tempStorage.snapshot(), baseServerUrl)
	}
}
