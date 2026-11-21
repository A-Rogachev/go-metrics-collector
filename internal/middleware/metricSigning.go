package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	loggers "github.com/A-Rogachev/go-metrics-collector/internal/logger"
	"github.com/labstack/echo/v5"
)

const signatureHeader string = "HashSHA256"

type responseBuffer struct {
	http.ResponseWriter
	body   bytes.Buffer
	status int
}

func (w *responseBuffer) WriteHeader(status int) {
	w.status = status
}

func (w *responseBuffer) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func MetricSigningMiddleware(logger loggers.Logger, secretKey *string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			request := c.Request()
			if request.Method == http.MethodPut {
				signHeaderValue, err := hex.DecodeString(request.Header.Get(signatureHeader))
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "request signature is required")
				}

				body, err := io.ReadAll(request.Body)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "invalid request signature format")
				}

				h := hmac.New(sha256.New, []byte(*secretKey))
				h.Write(body)
				sign := h.Sum(nil)
				if !hmac.Equal(sign, signHeaderValue) {
					return echo.NewHTTPError(http.StatusBadRequest, "invalid request signature")
				}
				request.Body = io.NopCloser(bytes.NewReader(body))
			}

			originalResponse := c.Response()
			buffered := &responseBuffer{
				ResponseWriter: originalResponse,
				status:         http.StatusOK,
			}
			c.SetResponse(buffered)

			err := next(c)
			if err != nil {
				c.SetResponse(originalResponse)
				return err
			}

			hmacResp := hmac.New(sha256.New, []byte(*secretKey))
			hmacResp.Write(buffered.body.Bytes())
			signature := hex.EncodeToString(hmacResp.Sum(nil))

			buffered.Header().Set(signatureHeader, signature)
			c.SetResponse(originalResponse)
			originalResponse.WriteHeader(buffered.status)

			if _, err := originalResponse.Write(buffered.body.Bytes()); err != nil {
				return err
			}
			return nil
		}
	}
}
