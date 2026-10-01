package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestMetricStorageCollect(t *testing.T) {
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
	_ = SendRequests(client, logger, snapshot, server.URL+"/update")

	assert.ElementsMatch(t, []string{
		"/update/gauge/Alloc/1.230000",
		"/update/counter/PollCount/2",
	}, fromAgent)
}

func TestSendMetric(t *testing.T) {
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
