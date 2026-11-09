package handler

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"strconv"

	loggers "github.com/A-Rogachev/go-metrics-collector/internal/logger"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
)

type DBHealthChecker interface {
	Ping(context.Context) error
}

func PingDatabase(c *echo.Context, logger loggers.Logger, checker DBHealthChecker) error {
	if err := checker.Ping(c.Request().Context()); err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			logger.Error().Msg(err.Error())
			return echo.NewHTTPError(
				http.StatusServiceUnavailable,
				"database ping timeout",
			)
		case errors.Is(err, context.Canceled):
			return echo.NewHTTPError(
				http.StatusRequestTimeout,
				"request canceled",
			)
		default:
			logger.Error().Msg(err.Error())
			return echo.NewHTTPError(
				http.StatusServiceUnavailable,
				"database unavailable",
			)
		}
	}
	return c.NoContent(http.StatusOK)
}

func GetMetricValue(c *echo.Context, s Storage) error {
	metricType := c.Param("metricType")
	_, ok := metricHandlers[metricType]
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, InvalidMetricTypeMsg)
	}
	key := c.Param("key")
	value, err := s.GetStringValue(metricType, key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, MetricNotFound)
	}
	return c.String(http.StatusOK, value)

}

func GetMetricValueJSON(c *echo.Context, s Storage) error {
	var askedMetric models.GetMetricRequest

	if err := c.Bind(&askedMetric); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, JSONParsingError)
	}
	// тип метрики (наличие хендлера) валидируется в сериализаторе oneof=gauge counter
	if err := c.Validate(askedMetric); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			return c.JSON(http.StatusBadRequest, map[string]any{
				"errors": getErrorResponse(&validationErrors),
			})
		}
		return echo.NewHTTPError(http.StatusBadRequest, "validation error")
	}

	value, err := s.GetStringValue(askedMetric.MType, askedMetric.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Metric key not found")
	}
	result, err := s.GetNumberValue(askedMetric.MType, value)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Metric value error")
	}
	response := models.Metric{
		ID:    askedMetric.ID,
		MType: askedMetric.MType,
	}
	switch value := result.(type) {
	case float64:
		response.Value = &value
	case int64:
		response.Delta = &value
	}
	return c.JSON(http.StatusOK, response)
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

func formatMetricValue(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func GetAllMetrics(c *echo.Context, s Storage) error {
	c.Response().Header().Set("content-type", "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	return metricsTemplate.Execute(c.Response(), s.Snapshot())
}
