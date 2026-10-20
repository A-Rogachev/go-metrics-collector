package storage

import (
	"fmt"
	"maps"
	"strconv"
	"sync"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

type MemStorage struct {
	mu       sync.Mutex
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

func (s *MemStorage) GetStringValue(metricType string, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch metricType {
	case models.Gauge:
		value, ok := s.gauges[key]
		if !ok {
			return "", fmt.Errorf("gauge %q not found", key)
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case models.Counter:
		value, ok := s.counters[key]
		if !ok {
			return "", fmt.Errorf("counter %q not found", key)
		}
		return strconv.FormatInt(value, 10), nil
	default:
		return "", fmt.Errorf("invalid metric type %q", metricType)
	}
}

func (s *MemStorage) GetNumberValue(metricType string, value string) (any, error) {
	var result any
	var err error

	switch metricType {
	case models.Gauge:
		result, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return float64(0.0), fmt.Errorf("error while converting metric")
		}
	case models.Counter:
		result, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return int64(0), fmt.Errorf("error while converting metric")
		}
	}
	return result, nil
}

func (s *MemStorage) Snapshot() models.MetricSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return models.MetricSnapshot{
		Gauges:   maps.Clone(s.gauges),
		Counters: maps.Clone(s.counters),
	}
}

func (s *MemStorage) LoadFromSnapshot(snapshot models.MetricSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges = snapshot.Gauges
	s.counters = snapshot.Counters
	return nil
}
