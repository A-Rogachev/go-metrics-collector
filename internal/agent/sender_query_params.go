package agent

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

type QueryParamsSender struct {
	client *http.Client
	logger *slog.Logger
	URL    string
}

func NewQueryParamsSender(client *http.Client, logger *slog.Logger, baseURL string) *QueryParamsSender {
	return &QueryParamsSender{
		client: client,
		logger: logger,
		URL:    baseURL,
	}
}

func (s *QueryParamsSender) SendMetrics(snapshot models.MetricSnapshot) map[string]int64 {
	sentCounters := make(map[string]int64, 0)
	for key, value := range snapshot.Gauges {
		url := fmt.Sprintf("%s/gauge/%s/%f", s.URL, key, value)
		if err := s.SendMetric(url); err != nil {
			s.logger.Debug(err.Error())
		}
	}
	for key, value := range snapshot.Counters {
		url := fmt.Sprintf("%s/counter/%s/%d", s.URL, key, value)
		if err := s.SendMetric(url); err != nil {
			s.logger.Debug(err.Error())
			continue
		}
		sentCounters[key] = value
	}
	s.logger.Info("Metrics sent to the server via query params")
	return sentCounters
}

func (s *QueryParamsSender) SendMetric(url string) error {
	response, err := s.client.Post(url, "text/plain", nil)
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
