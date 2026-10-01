package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
	mem_storage "github.com/A-Rogachev/go-metrics-collector/internal/storage"
)

const (
	InvalidMetricTypeMsg  = "Invalid metric type"
	InvalidMetricValueMsg = "Invalid metric value"
	MetricNotFound        = "Metric key not found"
)

type Storage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, value int64)
	GetStringValue(metricType string, key string) (string, error)
	Snapshot() mem_storage.MetricSnapshot
}

type metricHandler func(key string, rawValue string, storage Storage) error

var metricHandlers = map[string]metricHandler{
	models.Counter: handleCounterMetric,
	models.Gauge:   handleGaugeMetric,
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

func UpdateMetric(c *echo.Context, storage Storage) error {
	metricType := c.Param("metricType")
	key, rawValue := c.Param("key"), c.Param("value")
	handler, ok := metricHandlers[metricType]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricTypeMsg)
	}
	if err := handler(key, rawValue, storage); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricValueMsg)
	}
	return c.NoContent(http.StatusOK)
}

func GetMetricValue(c *echo.Context, storage Storage) error {
	metricType := c.Param("metricType")
	_, ok := metricHandlers[metricType]
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, InvalidMetricTypeMsg)
	}
	key := c.Param("key")
	value, err := storage.GetStringValue(metricType, key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Metric key not found")
	}
	return c.String(http.StatusOK, value)

}

var metricsTemplate = template.Must(template.New("metrics").Parse(`
<html>
<body>
{{ range $name, $value := .Gauges }}
{{ $name }}: {{ $value }}<br>
{{ end }}
{{ range $name, $value := .Counters }}
{{ $name }}: {{ $value }}<br>
{{ end }}
</body>
</html>`))

func GetAllMetrics(c *echo.Context, storage Storage) error {
	c.Response().WriteHeader(http.StatusOK)
	return metricsTemplate.Execute(c.Response(), storage.Snapshot())
}
