package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusHandler(t *testing.T) {
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
	type params struct {
		metricType string
		key        string
		value      string
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
				contentType: "text/plain; charset=utf-8",
				hasGauge:    false,
			},
		},
		{
			name:   "invalid gauge value test #3",
			url:    "/update/gauge/validKey/invalidValue",
			method: http.MethodPost,
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
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
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "invalid http method test #6",
			url:    "/update/counter/validKey/1232",
			method: http.MethodGet,
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	storage := NewMemStorage()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /update/{metricType}/{key}/{value}", func(w http.ResponseWriter, r *http.Request) {
		updateMetric(w, r, storage)
	})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.url, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, test.want.code, rec.Code, "statuc code doesnt match")
			assert.Equal(t, test.want.contentType, rec.Header().Get("Content-Type"), "content type doesnt match")

			if test.want.hasGauge {
				value, ok := storage.gauges[test.want.gaugeKey]
				assert.True(t, ok, "gauge key should be saved")
				assert.Equal(t, test.want.gaugeValue, value, "gauge value doesn't match")
			}

			if test.want.hasCounter {
				value, ok := storage.counters[test.want.counterKey]
				assert.True(t, ok, "counter key should be saved")
				assert.Equal(t, test.want.counterValue, value, "counter value doesn't match")
			}

		})
	}
}
