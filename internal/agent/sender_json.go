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

type JSONSender struct {
	client *http.Client
	logger *slog.Logger
	URL    string
}

func NewJSONSender(client *http.Client, logger *slog.Logger, baseURL string) *JSONSender {
	return &JSONSender{
		client: client,
		logger: logger,
		URL:    baseURL + "/",
	}
}

func (s *JSONSender) SendMetrics(snapshot models.MetricSnapshot) map[string]int64 {
	sentCounters := make(map[string]int64, 0)
	for key, value := range snapshot.Gauges {
		reqBody := models.Metric{
			ID:    key,
			MType: models.Gauge,
			Value: &value,
		}
		if err := s.SendJSONMetric(reqBody); err != nil {
			s.logger.Debug(err.Error())
		}
	}
	for key, value := range snapshot.Counters {
		reqBody := models.Metric{
			ID:    key,
			MType: models.Counter,
			Delta: &value,
		}
		if err := s.SendJSONMetric(reqBody); err != nil {
			s.logger.Debug(err.Error())
			continue
		}
		sentCounters[key] = value
	}
	s.logger.Info("Metrics sent to the server via JSON")
	return sentCounters
}

func (s *JSONSender) SendJSONMetric(body models.Metric) error {
	jsonBody, err := json.Marshal(body)
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
