package handler

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

const (
	InvalidMetricTypeMsg  = "Invalid metric type"
	InvalidMetricValueMsg = "Invalid metric value"
	MetricNotFound        = "Metric key not found"
	JSONParsingError      = "Error while json parsing"
)

type Storage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, value int64)
	GetStringValue(metricType string, key string) (string, error)
	Snapshot() models.MetricSnapshot
}

type metricHandler func(key string, rawValue string, storage Storage) error

type metricJSONHandler func(metric models.Metrics, storage Storage) error

var metricHandlers = map[string]metricHandler{
	models.Counter: handleCounterMetric,
	models.Gauge:   handleGaugeMetric,
}

var metricJSONHandlers = map[string]metricJSONHandler{
	models.Counter: handleCounterJSONMetric,
	models.Gauge:   handleGaugeJSONMetric,
}

func handleGaugeJSONMetric(metric models.Metrics, storage Storage) error {
	if metric.Value == nil {
		return fmt.Errorf("gauge value is required")
	}

	storage.SetGauge(metric.ID, *metric.Value)
	return nil
}

func handleCounterJSONMetric(metric models.Metrics, storage Storage) error {
	if metric.Delta == nil {
		return fmt.Errorf("counter delta is required")
	}

	storage.AddCounter(metric.ID, *metric.Delta)
	return nil
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

func formatMetricValue(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

var metricsTemplate = template.Must(template.New("metrics").Funcs(template.FuncMap{
	"formatValue": formatMetricValue,
}).Parse(`
<html>
<body>
{{ range $name, $value := .Gauges }}
{{ $name }}: {{ formatValue $value }}<br>
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

func UpdateMetricJSON(c *echo.Context, storage Storage) error {
	var newMetric models.Metrics

	if err := c.Bind(&newMetric); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, JSONParsingError)
	}
	if err := c.Validate(newMetric); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			errorsResponse := make([]ValidationErrorResponse, 0, len(validationErrors))
			for _, validationError := range validationErrors {
				errorsResponse = append(errorsResponse, ValidationErrorResponse{
					Field:   validationError.Field(),
					Tag:     validationError.Tag(),
					Value:   validationError.Value(),
					Message: validationError.Error(),
				})
			}
			return c.JSON(http.StatusBadRequest, map[string]any{
				"errors": errorsResponse,
			})
		}
	}
	handler, _ := metricJSONHandlers[newMetric.MType]
	if err := handler(newMetric, storage); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricValueMsg)
	}
	return c.NoContent(http.StatusOK)
}

func GetAllMetricsJSON(c *echo.Context, storage Storage) error {
	c.Response().WriteHeader(http.StatusOK)
	return metricsTemplate.Execute(c.Response(), storage.Snapshot())
}
