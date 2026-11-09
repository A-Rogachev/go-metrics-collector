package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	loggers "github.com/A-Rogachev/go-metrics-collector/internal/logger"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

func handleGaugeJSONMetric(metric models.Metric, storage Storage) error {
	if metric.Value == nil {
		return fmt.Errorf("gauge value is required")
	}

	storage.SetGauge(metric.ID, *metric.Value)
	return nil
}

func handleCounterJSONMetric(metric models.Metric, storage Storage) error {
	if metric.Delta == nil {
		return fmt.Errorf("counter delta is required")
	}

	storage.AddCounter(metric.ID, *metric.Delta)
	return nil
}

func handleCounterMetric(key string, rawValue string, storage Storage) (models.Metric, error) {
	delta, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		return models.Metric{}, err
	}
	metric := models.Metric{
		ID:    key,
		MType: models.Gauge,
		Delta: &delta,
	}
	storage.AddCounter(key, delta)
	return metric, nil
}

func handleGaugeMetric(key string, rawValue string, storage Storage) (models.Metric, error) {
	value, err := strconv.ParseFloat(rawValue, 64)
	if err != nil {
		return models.Metric{}, err
	}
	metric := models.Metric{
		ID:    key,
		MType: models.Gauge,
		Value: &value,
	}
	storage.SetGauge(key, value)
	return metric, nil
}

func UpdateMetric(
	c *echo.Context,
	s Storage,
	saver MetricSaver,
	logger loggers.Logger,
) error {
	metricType := c.Param("metricType")
	key, rawValue := c.Param("key"), c.Param("value")
	handler, ok := metricHandlers[metricType]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricTypeMsg)
	}
	metric, err := handler(key, rawValue, s)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricValueMsg)
	}
	if err := saver.SaveOnUpdate(c.Request().Context(), metric); err != nil {
		logger.Error().Msg(err.Error())
	}
	return c.NoContent(http.StatusOK)
}

func UpdateMetricJSON(
	c *echo.Context,
	s Storage,
	saver MetricSaver,
	logger loggers.Logger,
) error {
	var newMetric models.Metric

	if err := c.Bind(&newMetric); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, JSONParsingError)
	}
	// тип метрики (наличие хендлера) валидируется в сериализаторе oneof=gauge counter
	if err := c.Validate(newMetric); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			return c.JSON(http.StatusBadRequest, map[string]any{
				"errors": getErrorResponse(&validationErrors),
			})
		}
		return echo.NewHTTPError(http.StatusBadRequest, "validation error")
	}
	handler, _ := metricJSONHandlers[newMetric.MType]
	if err := handler(newMetric, s); err != nil {
		logger.Error().Msg(err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricValueMsg)
	}
	if err := saver.SaveOnUpdate(c.Request().Context(), newMetric); err != nil {
		logger.Error().Msg(err.Error())
	}
	return c.NoContent(http.StatusOK)
}

func BatchMetricUpdate(
	c *echo.Context,
	s Storage,
	saver MetricSaver,
	logger loggers.Logger,
) error {
	var newMetrics []models.Metric
	if err := c.Bind(&newMetrics); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, JSONParsingError)
	}
	var completeErrors []ValidationErrorResponse
	for _, m := range newMetrics {
		if err := c.Validate(m); err != nil {
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				completeErrors = append(
					completeErrors,
					getErrorResponse(&validationErrors)...,
				)
			} else {
				return echo.NewHTTPError(http.StatusBadRequest, "validation error")
			}
		}
	}
	if len(completeErrors) > 0 {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"errors": completeErrors,
		})
	}
	for _, m := range newMetrics {
		handler, _ := metricJSONHandlers[m.MType]
		if err := handler(m, s); err != nil {
			logger.Error().Msg(err.Error())
			return echo.NewHTTPError(http.StatusBadRequest, InvalidMetricValueMsg)
		}
	}
	if err := saver.SaveBatchOnUpdate(c.Request().Context(), newMetrics); err != nil {
		logger.Error().Msg(err.Error())
	}
	return c.NoContent(http.StatusOK)
}
