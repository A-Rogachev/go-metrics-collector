package middleware

import (
	"bytes"
	"io"
	"net/http"

	loggers "github.com/A-Rogachev/go-metrics-collector/internal/logger"
	"github.com/labstack/echo/v5"
)

const signatureHeader string = "HashSHA256"

type Signer interface {
	Sign([]byte) (string, error)
	Verify(data []byte, signature string) bool
}

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

func MetricSigningMiddleware(
	logger loggers.Logger,
	signer Signer,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			request := c.Request()
			if request.Method == http.MethodPost {
				headerName := http.CanonicalHeaderKey(signatureHeader)
				values, exists := request.Header[headerName]
				if exists {
					var signature string
					if len(values) > 0 {
						signature = values[0]
					}
					body, err := io.ReadAll(request.Body)
					if err != nil {
						return echo.NewHTTPError(
							http.StatusBadRequest,
							"request body error",
						)
					}
					if !signer.Verify(body, signature) {
						return echo.NewHTTPError(
							http.StatusBadRequest,
							"invalid request signature",
						)
					}
					request.Body = io.NopCloser(bytes.NewReader(body))
				}
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

			signature, _ := signer.Sign(buffered.body.Bytes())
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
