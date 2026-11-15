package middleware

import (
	"fmt"
	"net/http"
	"strings"

	loggers "github.com/A-Rogachev/go-metrics-collector/internal/logger"
	"github.com/labstack/echo/v5"
)

func MetricSigningMiddleware(logger loggers.Logger, secretKey *string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			request := c.Request()
			if !strings.Contains(request.Header.Get("Content-Encoding"), "gzip") {
				return next(c)
			}
			compReader, err := newCompressReader(request.Body)
			if err != nil {
				logger.Error().Msg(fmt.Sprintf("%s", err))
				return echo.NewHTTPError(http.StatusBadRequest, "invalid gzip request body")
			}
			request.Body = compReader
			request.Header.Del("Content-Encoding")
			defer request.Body.Close()
			return next(c)
		}
	}
}

// проверить на сервере если есть ключ и хидер, выдать 400 если не совпадает
// добавить в ответ также сумму ответа если есть ключ в конфиге
