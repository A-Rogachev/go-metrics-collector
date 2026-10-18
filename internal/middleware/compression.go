package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

type compressWriter struct {
	w           http.ResponseWriter
	zw          *gzip.Writer
	wroteHeader bool
	compress    bool
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{w: w}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Unwrap() http.ResponseWriter {
	return c.w
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if c.wroteHeader {
		return
	}
	c.wroteHeader = true
	contentType := c.Header().Get("Content-Type")
	c.compress = statusCode >=
		http.StatusOK && statusCode < http.StatusMultipleChoices && isCompressible(contentType)
	if c.compress {
		c.Header().Set("Content-Encoding", "gzip")
		c.Header().Add("Vary", "Accept-Encoding")
		c.Header().Del("Content-Length")
		// fmt.Println("compressed")
		c.zw = gzip.NewWriter(c.w)
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Write(data []byte) (int, error) {
	if !c.wroteHeader {
		if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", http.DetectContentType(data))
		}
		c.WriteHeader(http.StatusOK)
	}
	if c.compress {
		return c.zw.Write(data)
	}
	return c.w.Write(data)
}

func (c *compressWriter) Close() error {
	if c.zw == nil {
		return nil
	}
	return c.zw.Close()
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func isCompressible(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "application/json" ||
		mediaType == "text/plain" ||
		mediaType == "text/html"
}

func GzipRequestMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		request := c.Request()
		if !strings.Contains(request.Header.Get("Content-Encoding"), "gzip") {
			return next(c)
		}
		gzipReader, err := gzip.NewReader(request.Body)
		if err != nil {
			fmt.Println(err)
			return echo.NewHTTPError(http.StatusBadRequest, "invalid gzip request body")
		}
		request.Body = &compressReader{r: request.Body, zr: gzipReader}
		request.Header.Del("Content-Encoding")
		request.Header.Del("Content-Length")
		request.ContentLength = -1 // размер body неизвестен
		defer request.Body.Close()
		return next(c)
	}
}

func GzipResponseMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if !strings.Contains(c.Request().Header.Get("Accept-Encoding"), "gzip") {
			return next(c)
		}
		originalResponse := c.Response()
		compressedResponse := newCompressWriter(originalResponse)
		c.SetResponse(compressedResponse)

		err := next(c)
		if err != nil {
			c.Echo().HTTPErrorHandler(c, err)
			err = nil
		}
		closeErr := compressedResponse.Close()
		c.SetResponse(originalResponse)
		if closeErr != nil {
			return closeErr
		}
		return err
	}
}
