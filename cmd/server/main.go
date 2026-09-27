package main

import (
	"net/http"
	"strconv"
	"sync"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

type Storage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, value int64)
}

type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MemStorage) SetGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.gauges[name] = value
}

func (s *MemStorage) AddCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counters[name] += value
}

type metricHandler func(key string, rawValue string, storage Storage) error

var metricHandlers = map[string]metricHandler{
	Counter: handleCounterMetric,
	Gauge:   handleGaugeMetric,
}

func handleCounterMetric(key string, rawValue string, storage Storage) error {
	value, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		return err
	}
	storage.AddCounter(key, value)
	return nil
}

func handleGaugeMetric(key string, rawValue string, storage Storage) error {
	value, err := strconv.ParseFloat(rawValue, 64)
	if err != nil {
		return err
	}
	storage.SetGauge(key, value)
	return nil
}

func updateMetric(w http.ResponseWriter, req *http.Request, storage Storage) {
	metricType := req.PathValue("metricType")
	key, rawValue := req.PathValue("key"), req.PathValue("value")
	handler, ok := metricHandlers[metricType]
	if !ok {
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}
	if err := handler(key, rawValue, storage); err != nil {
		http.Error(w, "Invalid value", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func main() {
	storage := NewMemStorage()
	mux := http.NewServeMux()
	mux.HandleFunc(`POST /update/{metricType}/{key}/{value}`, func(w http.ResponseWriter, r *http.Request) {
		updateMetric(w, r, storage)
	})
	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}

}
