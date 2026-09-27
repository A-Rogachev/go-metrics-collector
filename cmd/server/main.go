package main

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/labstack/echo/v5"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

const (
	InvalidMetricTypeMsg  = "Invalid metric type"
	InvalidMetricValueMsg = "Invalid metric value"
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

func updateMetric(c *echo.Context, storage Storage) error {
	metricType := c.Param("metricType")
	key, rawValue := c.Param("key"), c.Param("value")
	handler, ok := metricHandlers[metricType]
	if !ok {
		c.Logger().Error(InvalidMetricTypeMsg, "metricType", metricType)
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricTypeMsg)
	}
	if err := handler(key, rawValue, storage); err != nil {
		c.Logger().Error(InvalidMetricValueMsg, "metricValue", rawValue)
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricValueMsg)
	}
	return c.NoContent(http.StatusOK)
}

func main() {
	e := echo.New()
	storage := NewMemStorage()

	e.POST("/update/:metricType/:key/:value", func(c *echo.Context) error {
		return updateMetric(c, storage)
	})
	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
