package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

const BatchSize = 50

type JSONBatchSender struct {
	client *http.Client
	logger *slog.Logger
	URL    string
}

func NewJSONBatchSender(client *http.Client, logger *slog.Logger, baseURL string) *JSONBatchSender {
	return &JSONBatchSender{
		client: client,
		logger: logger,
		URL:    baseURL + "/",
	}
}

func (s *JSONBatchSender) SendMetrics(snapshot models.MetricSnapshot) map[string]int64 {
	sentCounters := make(map[string]int64, 0)
	var metricsToSend []models.Metric
	for key, value := range snapshot.Gauges {
		metricsToSend = append(
			metricsToSend,
			models.Metric{
				ID:    key,
				MType: models.Gauge,
				Value: &value,
			},
		)
	}
	for key, value := range snapshot.Counters {
		metricsToSend = append(
			metricsToSend,
			models.Metric{
				ID:    key,
				MType: models.Counter,
				Delta: &value,
			},
		)
		sentCounters[key] = value
	}
	if len(metricsToSend) > 0 {
		err := s.sendBatches(metricsToSend)
		if err != nil {
			s.logger.Debug(err.Error())
			return make(map[string]int64, 0)
		} else {
			s.logger.Info("Metrics sent to the server")
			return sentCounters
		}
	}
	s.logger.Info("No metrics to send")
	return sentCounters
}

func (s *JSONBatchSender) sendBatches(metrics []models.Metric) error {
	for start := 0; start < len(metrics); start += BatchSize {
		end := start + BatchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		if err := s.sendJSONMetricBatch(metrics[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s *JSONBatchSender) sendJSONMetricBatch(metrics []models.Metric) error {
	jsonBody, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	var compressedBody bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBody)
	if _, err := gzipWriter.Write(jsonBody); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, s.URL, &compressedBody)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", response.StatusCode)
	}
	_, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		return err
	}
	return nil
}
