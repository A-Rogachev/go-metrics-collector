package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"

	handlers "github.com/A-Rogachev/go-metrics-collector/internal/handler"
	"github.com/A-Rogachev/go-metrics-collector/internal/middleware"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
	services "github.com/A-Rogachev/go-metrics-collector/internal/service"

	int_storage "github.com/A-Rogachev/go-metrics-collector/internal/storage"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

const skipTestMessage string = "=================== skipping test in short mode"

func TestUpdateMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}
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
	storage := int_storage.NewMemStorage()
	fileStorage := int_storage.NewFileStorage("")
	appLogger := middleware.ZerologAdapter{Logger: getLogger("debug")}
	saver := services.NewMetricSaver(storage, fileStorage, time.Duration(2), appLogger)
	e := echo.New()
	e.POST("/update/:metricType/:key/:value", func(c *echo.Context) error {
		return handlers.UpdateMetric(c, storage, saver, appLogger)
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
	if testing.Short() {
		t.Skip(skipTestMessage)
	}

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
	storage := int_storage.NewMemStorage()
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
	if testing.Short() {
		t.Skip(skipTestMessage)
	}

	var (
		gaugeKey           = "Alloc"
		gaugeValue         = 1.23
		counterKey         = "pollCount"
		counterValue int64 = 3

		metricsURL = "/"
	)

	storage := int_storage.NewMemStorage()
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

func TestUpdateMetricJSON(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}

	var (
		gaugeValue   = float64(1.23)
		counterValue = int64(3)
		URL          = "/update/"
	)
	type expected struct {
		statusCode   int
		gaugeValue   float64
		counterValue int64
	}
	tests := []struct {
		name        string
		requestBody models.Metric
		isGauge     bool
		isCounter   bool
		expected    expected
		contentType string
	}{
		{
			name: "update valid gauge metric #1",
			requestBody: models.Metric{
				ID:    "gaugeKey1",
				MType: "gauge",
				Value: &gaugeValue,
			},
			isGauge: true,
			expected: expected{
				statusCode: http.StatusOK,
				gaugeValue: float64(1.23),
			},
			contentType: "application/json",
		},
		{
			name: "update valid counter metric #2",
			requestBody: models.Metric{
				ID:    "counterKey1",
				MType: "counter",
				Delta: &counterValue,
			},
			isCounter: true,
			expected: expected{
				statusCode:   http.StatusOK,
				counterValue: int64(3),
			},
			contentType: "application/json",
		},
		{
			name: "update valid counter metric again #3",
			requestBody: models.Metric{
				ID:    "counterKey1",
				MType: "counter",
				Delta: &counterValue,
			},
			isCounter: true,
			expected: expected{
				statusCode:   http.StatusOK,
				counterValue: int64(counterValue * 2),
			},
			contentType: "application/json",
		},
		{
			name: "invalid request body (type) #4",
			requestBody: models.Metric{
				ID:    "counterKey1",
				MType: "counter!",
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
		},
		{
			name: "invalid content type #5",
			requestBody: models.Metric{
				ID:    "counterKey1",
				MType: "counter",
				Delta: &counterValue,
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
			contentType: "text/plain",
		},
	}

	storage := int_storage.NewMemStorage()
	fileStorage := int_storage.NewFileStorage("")
	appLogger := middleware.ZerologAdapter{Logger: getLogger("debug")}
	saver := services.NewMetricSaver(storage, fileStorage, time.Duration(2), appLogger)
	e := echo.New()
	e.Validator = &handlers.MetricValidator{Validator: validator.New()}
	e.POST("update/", func(c *echo.Context) error {
		return handlers.UpdateMetricJSON(c, storage, saver, appLogger)
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(test.requestBody)
			request := httptest.NewRequest(http.MethodPost, URL, bytes.NewBuffer(body))
			request.Header.Set("Content-Type", test.contentType)

			e.ServeHTTP(w, request)

			assert.Equal(t, test.expected.statusCode, w.Code)
			if test.isGauge {
				stringValue := strconv.FormatFloat(test.expected.gaugeValue, 'f', -1, 64)
				valueFromStorage, _ := storage.GetNumberValue(models.Gauge, stringValue)
				assert.Equal(t, valueFromStorage, test.expected.gaugeValue)
			}
			if test.isCounter {
				stringValue := strconv.FormatInt(test.expected.counterValue, 10)
				valueFromStorage, _ := storage.GetNumberValue(models.Counter, stringValue)
				assert.Equal(t, valueFromStorage, test.expected.counterValue)
			}
		})
	}

}

func TestGetMetricValueJSON(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}

	var (
		gaugeKey           = "Alloc"
		gaugeValue         = 1.23
		counterValue int64 = 3
		counterKey         = "pollCount"

		URL = "/value/"
	)
	type expected struct {
		statusCode int
		hasDelta   bool
		hasValue   bool
		value      float64
		delta      int64
	}
	tests := []struct {
		name        string
		requestBody models.GetMetricRequest
		expected    expected
		contentType string
	}{
		{
			name: "valid request and metric exists #1",
			requestBody: models.GetMetricRequest{
				ID:    gaugeKey,
				MType: "gauge",
			},
			expected: expected{
				statusCode: http.StatusOK,
				hasValue:   true,
				value:      gaugeValue,
			},
			contentType: "application/json",
		},
		{
			name: "valid request and metric exists #2",
			requestBody: models.GetMetricRequest{
				ID:    counterKey,
				MType: "counter",
			},
			expected: expected{
				statusCode: http.StatusOK,
				hasDelta:   true,
				delta:      int64(counterValue),
			},
			contentType: "application/json",
		},
		{
			name: "invalid metric type #3",
			requestBody: models.GetMetricRequest{
				ID:    counterKey,
				MType: "counterX",
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
		},
		{
			name: "invalid metric key #4",
			requestBody: models.GetMetricRequest{
				ID:    counterKey + "!",
				MType: "counter",
			},
			expected: expected{
				statusCode: http.StatusNotFound,
			},
			contentType: "application/json",
		},
		{
			name: "invalid request (missed ID field) #5",
			requestBody: models.GetMetricRequest{
				MType: "counter",
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
		},
		{
			name: "invalid request content type #6",
			requestBody: models.GetMetricRequest{
				MType: "counter",
			},
			expected: expected{
				statusCode: http.StatusBadRequest,
			},
			contentType: "plain/text",
		},
	}

	storage := int_storage.NewMemStorage()
	storage.SetGauge(gaugeKey, gaugeValue)
	storage.AddCounter(counterKey, counterValue)

	e := echo.New()
	e.Validator = &handlers.MetricValidator{Validator: validator.New()}
	e.POST(URL, func(c *echo.Context) error {
		return handlers.GetMetricValueJSON(c, storage)
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(test.requestBody)
			request := httptest.NewRequest(http.MethodPost, URL, bytes.NewBuffer(body))
			request.Header.Set("Content-Type", test.contentType)

			e.ServeHTTP(w, request)
			res := w.Result()
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			responseData := new(models.Metric)
			json.Unmarshal(resBody, &responseData)

			assert.NoError(t, err)
			assert.Equal(t, test.expected.statusCode, res.StatusCode, "status code doesnt match")
			assert.Equal(t, res.Header.Get("Content-type"), "application/json")
			if test.expected.hasDelta {
				assert.Equal(t, test.expected.delta, *responseData.Delta)
			}
			if test.expected.hasValue {
				assert.Equal(t, test.expected.value, *responseData.Value)
			}
		})
	}
}
