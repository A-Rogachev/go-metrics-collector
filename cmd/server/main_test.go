package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	handlers "github.com/A-Rogachev/go-metrics-collector/internal/handler"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
	mem_storage "github.com/A-Rogachev/go-metrics-collector/internal/storage"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMetrics(t *testing.T) {
	type want struct {
		code        int
		contentType string

		gaugeKey   string
		gaugeValue float64
		hasGauge   bool

		counterKey   string
		counterValue int64
		hasCounter   bool
	}
	tests := []struct {
		name   string
		url    string
		method string
		want   want
	}{
		{
			name:   "correct gauge metric test #1",
			url:    "/update/gauge/Alloc/1.23",
			method: http.MethodPost,
			want: want{
				code:        http.StatusOK,
				contentType: "",
				hasGauge:    true,
				gaugeKey:    "Alloc",
				gaugeValue:  float64(1.23),
			},
		},
		{
			name:   "gauge key missing test #2",
			url:    "/update/gauge/1.23",
			method: http.MethodPost,
			want: want{
				code:        http.StatusNotFound,
				contentType: "application/json",
				hasGauge:    false,
			},
		},
		{
			name:   "invalid gauge value test #3",
			url:    "/update/gauge/validKey/invalidValue",
			method: http.MethodPost,
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
				hasGauge:    false,
			},
		},
		{
			name:   "valid counter value test #4",
			url:    "/update/counter/validKey/123",
			method: http.MethodPost,
			want: want{
				code:         http.StatusOK,
				contentType:  "",
				hasCounter:   true,
				counterKey:   "validKey",
				counterValue: int64(123),
			},
		},
		{
			name:   "invalid counter value test #5",
			url:    "/update/counter/validKey/1232!",
			method: http.MethodPost,
			want: want{
				code:        http.StatusBadRequest,
				contentType: "application/json",
			},
		},
		{
			name:   "invalid http method test #6",
			url:    "/update/counter/validKey/1232",
			method: http.MethodGet,
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "application/json",
			},
		},
	}
	storage := mem_storage.NewMemStorage()
	e := echo.New()
	e.POST("/update/:metricType/:key/:value", func(c *echo.Context) error {
		return handlers.UpdateMetric(c, storage)
	})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			e.ServeHTTP(w, httptest.NewRequest(test.method, test.url, nil))
			res := w.Result()

			assert.Equal(t, test.want.code, res.StatusCode, "statuc code doesnt match")
			assert.Equal(t, test.want.contentType, w.Header().Get("Content-Type"), "content type doesnt match")

			if test.want.hasGauge {
				value, err := storage.GetStringValue(models.Gauge, test.want.gaugeKey)
				assert.NoError(t, err, "gauge key should be saved")
				assert.Equal(t, strconv.FormatFloat(test.want.gaugeValue, 'f', -1, 64), value, "gauge value doesn't match")
			}

			if test.want.hasCounter {
				value, err := storage.GetStringValue(models.Counter, test.want.counterKey)
				assert.NoError(t, err, "counter key should be saved")
				assert.Equal(t, strconv.FormatInt(test.want.counterValue, 10), value, "counter value doesn't match")
			}

		})
	}
}

func TestGetMetricValue(t *testing.T) {

	var (
		gaugeValue         = 1.23
		gaugeKey           = "Alloc"
		counterValue int64 = 3
		counterKey         = "pollCount"
	)

	type want struct {
		code     int
		hasValue bool
		value    string
	}
	tests := []struct {
		name string
		url  string
		want want
	}{
		{
			name: "existing gauge metric #1",
			url:  "/value/gauge/" + gaugeKey,
			want: want{
				code:     http.StatusOK,
				hasValue: true,
				value:    strconv.FormatFloat(gaugeValue, 'f', -1, 64),
			},
		},
		{
			name: "not existing gauge metric #2",
			url:  "/value/gauge/notExists",
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name: "existing counter metric #3",
			url:  "/value/counter/" + counterKey,
			want: want{
				code:     http.StatusOK,
				hasValue: true,
				value:    strconv.FormatInt(counterValue, 10),
			},
		},
		{
			name: "url without metric key #4",
			url:  "/value/counter/",
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name: "url with invalid metric type #4",
			url:  "/value/keyNotExists/" + gaugeKey,
			want: want{
				code: http.StatusNotFound,
			},
		},
	}
	storage := mem_storage.NewMemStorage()
	storage.SetGauge(gaugeKey, gaugeValue)
	storage.AddCounter(counterKey, counterValue)
	e := echo.New()
	e.GET("value/:metricType/:key", func(c *echo.Context) error {
		return handlers.GetMetricValue(c, storage)
	})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, test.url, nil))
			res := w.Result()

			assert.Equal(t, test.want.code, res.StatusCode, "statuc code doesnt match")
			if test.want.hasValue {
				defer res.Body.Close()
				resBody, _ := io.ReadAll(res.Body)
				assert.Equal(t, test.want.value, string(resBody))
			}
		})
	}
}

func TestGetAllMetrics(t *testing.T) {

	var (
		gaugeKey           = "Alloc"
		gaugeValue         = 1.23
		counterKey         = "pollCount"
		counterValue int64 = 3

		metricsURL = "/"
	)

	storage := mem_storage.NewMemStorage()
	storage.SetGauge(gaugeKey, gaugeValue)
	storage.AddCounter(counterKey, counterValue)

	e := echo.New()
	e.GET(metricsURL, func(c *echo.Context) error {
		return handlers.GetAllMetrics(c, storage)
	})

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, metricsURL, nil))
	res := w.Result()
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode, "statuc code doesnt match")
	assert.Contains(t, string(resBody), fmt.Sprint(gaugeKey, ": ", gaugeValue))
	assert.Contains(t, string(resBody), fmt.Sprint(counterKey, ": ", counterValue))
}
