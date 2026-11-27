package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

const BatchSize = 10

type JSONBatchSender struct {
	client    *http.Client
	logger    *slog.Logger
	URL       string
	signer    Signer
	batchSize int
}

func NewJSONBatchSender(
	client *http.Client,
	logger *slog.Logger,
	baseURL string,
	signer Signer,
) *JSONBatchSender {
	return &JSONBatchSender{
		client:    client,
		logger:    logger,
		URL:       baseURL + "/",
		signer:    signer,
		batchSize: BatchSize,
	}
}

func (s *JSONBatchSender) prepareMetrics(snapshot models.MetricSnapshot) []models.Metric {
	var metricsToSend []models.Metric
	for k, v := range snapshot.Gauges {
		metricsToSend = append(
			metricsToSend,
			models.Metric{ID: k, MType: models.Gauge, Value: &v},
		)
	}
	for k, v := range snapshot.Counters {
		metricsToSend = append(
			metricsToSend,
			models.Metric{ID: k, MType: models.Counter, Delta: &v},
		)
	}
	return metricsToSend
}

func (s *JSONBatchSender) SendMetrics(
	snapshot models.MetricSnapshot,
	poolSize int,
) map[string]int64 {
	sentCounters := make(map[string]int64, 0)
	metricsToSend := s.prepareMetrics(snapshot)
	if len(metricsToSend) == 0 {
		s.logger.Info("No metrics to send")
		return sentCounters
	}
	sentCounters = s.sendMetricsWithPool(metricsToSend, poolSize)
	return sentCounters

}

func (s *JSONBatchSender) sendMetricsWithPool(
	metrics []models.Metric,
	poolSize int,
) map[string]int64 {
	sentCounters := make(map[string]int64, 0)
	batchCh := make(chan []models.Metric, poolSize)
	resultCh := make(chan map[string]int64, poolSize)

	var workers sync.WaitGroup

	for i := 0; i < poolSize; i++ {
		workerID := i
		workers.Add(1)
		go func() {
			defer workers.Done()
			s.worker(batchCh, resultCh, workerID)
		}()
	}
	go s.splitMetrics(metrics, batchCh)
	go func() {
		workers.Wait()
		close(resultCh)
	}()

	for counters := range resultCh {
		for k, v := range counters {
			sentCounters[k] += v
		}
	}
	return sentCounters
}

func (s *JSONBatchSender) worker(
	chIn <-chan []models.Metric,
	chOut chan<- map[string]int64,
	workerID int,
) {
	for metrics := range chIn {
		sentCounters := s.sendOneBatch(metrics, workerID)
		chOut <- sentCounters
	}
}

// разбивает метрики по батчам для параллельной отправки воркерами
func (s *JSONBatchSender) splitMetrics(
	data []models.Metric,
	ch chan<- []models.Metric,
) {
	defer close(ch)
	for start := 0; start < len(data); start += s.batchSize {
		end := start + s.batchSize
		if end > len(data) {
			end = len(data)
		}
		ch <- data[start:end]
	}
}

func (s *JSONBatchSender) sendOneBatch(metrics []models.Metric, workerID int) map[string]int64 {
	sentCounters := make(map[string]int64)
	jsonBody, err := json.Marshal(metrics)
	if err != nil {
		s.logger.Debug(err.Error())
		return sentCounters
	}

	var compressedBody bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBody)
	if _, err := gzipWriter.Write(jsonBody); err != nil {
		s.logger.Debug(err.Error())
		return sentCounters
	}
	if err := gzipWriter.Close(); err != nil {
		s.logger.Debug(err.Error())
		return sentCounters
	}

	request, err := http.NewRequest(http.MethodPost, s.URL, &compressedBody)
	if err != nil {
		s.logger.Debug(err.Error())
		return sentCounters
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")

	metricSign, err := s.signer.SignMetric(jsonBody)
	if err == nil {
		request.Header.Set("HashSHA256", metricSign)
	}

	response, err := s.client.Do(request)
	if err != nil {
		s.logger.Debug(err.Error())
		return sentCounters
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status: %d", response.StatusCode)
		s.logger.Debug(err.Error())
		return sentCounters
	}
	_, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		s.logger.Debug(err.Error())
		return sentCounters
	}
	s.logger.Info(fmt.Sprintf("Metrics sent to the server (worker %d)", workerID+1))

	// в случае успешного запроса возвращается мапа с переданными счетчиками
	for _, m := range metrics {
		if m.MType == models.Counter {
			sentCounters[m.ID] = *m.Delta
		}
	}
	return sentCounters
}

func (s *JSONBatchSender) SendMetricsPeriodically(
	shapshotChan <-chan models.MetricSnapshot,
	countersChan chan<- map[string]int64,
	poolSize int,
) {
	for {
		metrics := <-shapshotChan
		metricsToSend := s.prepareMetrics(metrics)
		if len(metricsToSend) > 0 {
			sentCounters := s.sendMetricsWithPool(metricsToSend, poolSize)
			countersChan <- sentCounters
		}
	}
}
