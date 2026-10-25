package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"

	"github.com/stretchr/testify/assert"
)

const skipTestMessage string = "=================== skipping test in short mode"

func TestMetricStorageCollect(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}
	storage := AgentStorage{
		gauges:   make(map[string]float64),
		counters: map[string]int64{"PollCount": int64(0)},
	}
	pollInterval, polling := 1, false

	storage.collect(polling, pollInterval)

	assert.Equal(t, storage.counters["PollCount"], int64(1), "PollCount should be 1")
	assert.Greater(t, len(storage.gauges), 0, "gauges count greaater than 0")
}

func TestMetricStorageSnapshot(t *testing.T) {
	storage := AgentStorage{
		gauges:   map[string]float64{"Alloc": 123456},
		counters: map[string]int64{"PollCount": int64(1)},
	}
	storageSnapshot := storage.snapshot()
	assert.Equal(t, storageSnapshot.Gauges["Alloc"], float64(123456), "Alloc should be 123456")
	assert.Equal(t, storageSnapshot.Counters["PollCount"], int64(1), "PollCount should be 1")
}

func TestSendRequests(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}

	var fromAgent []string // пути запросов от агента к серверу

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromAgent = append(fromAgent, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := server.Client()
	snapshot := models.MetricSnapshot{
		Gauges:   map[string]float64{"Alloc": 1.23},
		Counters: map[string]int64{"PollCount": 2},
	}

	logger := getLogger("info")
	_ = sendRequests(client, logger, snapshot, server.URL+"/update")

	assert.ElementsMatch(t, []string{
		"/update/gauge/Alloc/1.230000",
		"/update/counter/PollCount/2",
	}, fromAgent)
}

func TestSendMetric(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}

	tests := []struct {
		name        string
		serverCode  int
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "successful request test #1",
			serverCode:  http.StatusOK,
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name:        "bad request test #2",
			serverCode:  http.StatusBadRequest,
			wantSuccess: false,
			wantErr:     true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.serverCode)
			}))
			defer server.Close()

			client := server.Client()
			err := sendMetric(client, server.URL+"/update/gauge/1/2")

			assert.Equal(t, test.wantErr, err != nil, "error while sending metric")
		})
	}
}

func TestSendJSONMetric(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}
	var gaugeValue = float64(1.23)
	tests := []struct {
		name        string
		serverCode  int
		wantErr     bool
		requestBody models.Metric
	}{
		{
			name:       "successful JSON request test #1",
			serverCode: http.StatusOK,
			requestBody: models.Metric{
				ID:    "gaugeKey1",
				MType: "gauge",
				Value: &gaugeValue,
			},
		},
		{
			name:       "invalid JSON body test #2",
			serverCode: http.StatusBadRequest,
			wantErr:    true,
			requestBody: models.Metric{
				ID:    "gaugeKey1",
				MType: "gauge",
				Value: &gaugeValue,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.serverCode)
			}))
			defer server.Close()

			client := server.Client()
			err := sendJSONMetric(client, server.URL+"/update/", test.requestBody)

			assert.Equal(t, test.wantErr, err != nil, "error while sending metric")
		})
	}
}

func TestSendJSONRequests(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}

	var received []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		gzipReader, err := gzip.NewReader(r.Body)
		assert.NoError(t, err)
		defer gzipReader.Close()

		body, err := io.ReadAll(gzipReader)
		assert.NoError(t, err)

		// NOTE: реализация без gzip
		// body, err := io.ReadAll(r.Body)
		// assert.NoError(t, err)

		received = append(received, string(body))
		w.WriteHeader(http.StatusOK)

		// NOTE:проверка заголовков сжатия gzip
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
	}))
	defer server.Close()

	client := server.Client()
	snapshot := models.MetricSnapshot{
		Gauges:   map[string]float64{"Alloc": 1.23},
		Counters: map[string]int64{"PollCount": 2},
	}
	expected := []string{
		`{"id":"Alloc","type":"gauge","value":1.23}`,
		`{"id":"PollCount","type":"counter","delta":2}`,
	}
	logger := getLogger("info")
	_ = sendJSONRequests(client, logger, snapshot, server.URL+"/update/")

	assert.JSONEq(t, expected[0], received[0])
	assert.JSONEq(t, expected[1], received[1])

}
