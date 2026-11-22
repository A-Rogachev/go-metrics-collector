package main

import (
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	agent "github.com/A-Rogachev/go-metrics-collector/internal/agent"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"

	"github.com/stretchr/testify/assert"
)

const skipTestMessage string = "=================== skipping test in short mode"
const signatureHeader string = "HashSHA256"
const signatureSecretKey string = "secret_key"

func TestMetricStorageCollect(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}
	storage := agent.NewAgentStorage()
	pollInterval, polling := 1, false

	storage.Collect(polling, time.Duration(pollInterval))
	gaugesCount, _ := storage.GetMetricsCount()
	counterValue, ok := storage.GetCounter("PollCount")

	assert.Equal(t, ok, true, "value wasnt recorded")
	assert.Equal(t, int64(1), counterValue, "PollCount should be 1")
	assert.Greater(t, gaugesCount, 0, "gauges count greater than 0")
}

func TestMetricStorageSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}
	storage := agent.NewAgentStorage()
	storage.SetGauge("Alloc", 123456)
	storage.AddCounter("PollCount", int64(1))
	storageSnapshot := storage.Snapshot()
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
	signer := agent.NewMetricSigner(nil, sha256.New)
	sender := agent.NewQueryParamsSender(client, logger, server.URL+"/update", signer)
	_ = sender.SendMetrics(snapshot)

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
			logger := getLogger("info")
			signer := agent.NewMetricSigner(nil, sha256.New)
			sender := agent.NewQueryParamsSender(client, logger, server.URL+"/update/gauge/1/2", signer)
			err := sender.SendMetric(sender.URL + "/update/gauge/1/2")

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
			logger := getLogger("info")
			signer := agent.NewMetricSigner(nil, sha256.New)
			sender := agent.NewJSONSender(client, logger, server.URL, signer)
			err := sender.SendJSONMetric(test.requestBody)

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
		defer r.Body.Close()
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
	signer := agent.NewMetricSigner(nil, sha256.New)
	sender := agent.NewJSONSender(client, logger, server.URL, signer)
	_ = sender.SendMetrics(snapshot)

	assert.JSONEq(t, expected[0], received[0])
	assert.JSONEq(t, expected[1], received[1])

}

func TestSendSha256Sum(t *testing.T) {
	if testing.Short() {
		t.Skip(skipTestMessage)
	}
	secretKey := signatureSecretKey
	tests := []struct {
		name      string
		algorithm func() hash.Hash
		isValid   bool
	}{
		{
			name:      "signature is valid #1",
			algorithm: sha256.New,
			isValid:   true,
		},
		{
			name:      "algorithm doesnt fit #2",
			algorithm: sha512.New,
			isValid:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gzipReader, _ := gzip.NewReader(r.Body)
				defer r.Body.Close()
				defer gzipReader.Close()
				signHeaderValue, _ := hex.DecodeString(r.Header.Get(signatureHeader))
				body, _ := io.ReadAll(gzipReader)

				h := hmac.New(sha256.New, []byte(secretKey))
				h.Write(body)
				sign := h.Sum(nil)

				assert.Equal(t, hmac.Equal(sign, signHeaderValue), test.isValid, "sha256 hash sum is not valid")
			}))

			defer server.Close()

			client := server.Client()
			snapshot := models.MetricSnapshot{
				Gauges:   map[string]float64{"Alloc": 1.23},
				Counters: map[string]int64{"PollCount": 2},
			}
			logger := getLogger("info")
			signer := agent.NewMetricSigner(&secretKey, test.algorithm)
			sender := agent.NewJSONBatchSender(client, logger, server.URL, signer)
			_ = sender.SendMetrics(snapshot)

		})
	}
}
