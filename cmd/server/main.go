package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	MetricNotFound        = "Metric key not found"
)

////////////////////////////////////////////////////////////////////

type Storage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, value int64)
	GetStringValue(metricType string, key string) (string, error)
	RenderMetricsToHTML() string
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

func (s *MemStorage) GetStringValue(metricType string, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch metricType {
	case Gauge:
		value, ok := s.gauges[key]
		if !ok {
			return "", fmt.Errorf("gauge %q not found", key)
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case Counter:
		value, ok := s.counters[key]
		if !ok {
			return "", fmt.Errorf("counter %q not found", key)
		}
		return strconv.FormatInt(value, 10), nil
	default:
		return "", fmt.Errorf("invalid metric type %q", metricType)
	}
}

func (s *MemStorage) RenderMetricsToHTML() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var b strings.Builder
	b.WriteString("<html><body>")
	for name, value := range s.gauges {
		b.WriteString(name)
		b.WriteString(": ")
		b.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
		b.WriteString("<br>")
	}
	for name, value := range s.counters {
		b.WriteString(name)
		b.WriteString(": ")
		b.WriteString(strconv.FormatInt(value, 10))
		b.WriteString("<br>")
	}
	b.WriteString("</body></html>")
	return b.String()
}

////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////

func updateMetric(c *echo.Context, storage Storage) error {
	metricType := c.Param("metricType")
	key, rawValue := c.Param("key"), c.Param("value")
	handler, ok := metricHandlers[metricType]
	if !ok {
		c.Logger().Error(InvalidMetricTypeMsg, "metricType", metricType)
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricTypeMsg)
	}
	if err := handler(key, rawValue, storage); err != nil {
		c.Logger().Error(InvalidMetricValueMsg, "value", rawValue)
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricValueMsg)
	}
	return c.NoContent(http.StatusOK)
}

func getMetricValue(c *echo.Context, storage Storage) error {
	metricType := c.Param("metricType")
	_, ok := metricHandlers[metricType]
	if !ok {
		c.Logger().Error(InvalidMetricTypeMsg, "metricType", metricType)
		return echo.NewHTTPError(http.StatusNotFound, InvalidMetricTypeMsg)
	}
	key := c.Param("key")
	value, err := storage.GetStringValue(metricType, key)
	if err != nil {
		c.Logger().Error(MetricNotFound, "key", key)
		return echo.NewHTTPError(http.StatusNotFound, "Metric key not found")
	}
	return c.String(http.StatusOK, value)

}

func getAllMetrics(c *echo.Context, storage Storage) error {
	return c.HTML(http.StatusOK, storage.RenderMetricsToHTML())
}

func main() {
	e := echo.New()
	storage := NewMemStorage()

	e.POST("/update/:metricType/:key/:value", func(c *echo.Context) error {
		return updateMetric(c, storage)
	})
	e.GET("value/:metricType/:key", func(c *echo.Context) error {
		return getMetricValue(c, storage)
	})
	e.GET("/", func(c *echo.Context) error {
		return getAllMetrics(c, storage)
	})
	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
