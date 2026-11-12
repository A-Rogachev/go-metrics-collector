package handler

import (
	"context"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

const (
	InvalidMetricTypeMsg  = "Invalid metric type"
	InvalidMetricValueMsg = "Invalid metric value"
	MetricNotFound        = "Metric key not found"
	JSONParsingError      = "Error while json parsing"
	UnknownError          = "Unknown error. Ask the administrator"
)

type Storage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, value int64)
	GetStringValue(metricType string, key string) (string, error)
	GetNumberValue(metricType string, value string) (any, error)
	Snapshot() models.MetricSnapshot
}

type MetricSaver interface {
	SaveOnUpdate(ctx context.Context, metric models.Metric) error
	SaveBatchOnUpdate(ctx context.Context, metrics []models.Metric) error
}

type metricHandler func(key string, rawValue string, storage Storage) (models.Metric, error)

type metricJSONHandler func(metric models.Metric, storage Storage) error

var metricHandlers = map[string]metricHandler{
	models.Counter: handleCounterMetric,
	models.Gauge:   handleGaugeMetric,
}

var metricJSONHandlers = map[string]metricJSONHandler{
	models.Counter: handleCounterJSONMetric,
	models.Gauge:   handleGaugeJSONMetric,
}
