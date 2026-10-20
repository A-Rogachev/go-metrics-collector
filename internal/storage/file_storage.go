package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

type FileStorage struct {
	filePath      string
	storeInterval time.Duration
	saveMutex     sync.Mutex
	Logger        any
}

type SnapshotProvider interface {
	Snapshot() models.MetricSnapshot
}

func NewFileStorage(filePath string, storeInterval int) *FileStorage {
	return &FileStorage{
		filePath:      filePath,
		storeInterval: time.Second * time.Duration(storeInterval),
	}
}

func (fs *FileStorage) LoadMetricsFromFile() (models.MetricSnapshot, error) {
	file, err := os.OpenFile(fs.filePath, os.O_RDONLY, 0444)
	var resultSnapshot = models.MetricSnapshot{
		Gauges:   map[string]float64{},
		Counters: map[string]int64{},
	}
	if err != nil {
		return resultSnapshot, err
	}
	defer file.Close()

	var metrics []models.Metric
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metrics); err != nil {
		if err == io.EOF {
			return resultSnapshot, nil
		}
		return resultSnapshot, err
	}
	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			resultSnapshot.Gauges[metric.ID] = *metric.Value
		case models.Counter:
			resultSnapshot.Counters[metric.ID] = *metric.Delta
		default:
			continue
		}
	}
	return resultSnapshot, nil
}

func (fs *FileStorage) PeriodicSaveToFile(storage SnapshotProvider) {
	if fs.storeInterval == time.Duration(0) {
		return
	}
	for {
		time.Sleep(fs.storeInterval)
		snapshot := storage.Snapshot()
		err := fs.UpdateFromSnapshot(snapshot, false)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (fs *FileStorage) UpdateFromSnapshot(snapshot models.MetricSnapshot, checkConfig bool) error {
	if checkConfig {
		if fs.storeInterval != time.Duration(0) {
			return nil
		}
	}
	fs.saveMutex.Lock()
	defer fs.saveMutex.Unlock()
	metrics := make([]models.Metric, 0)
	for key, value := range snapshot.Gauges {
		metrics = append(metrics, models.Metric{
			ID:    key,
			MType: models.Gauge,
			Value: &value,
		})
	}
	for key, value := range snapshot.Counters {
		metrics = append(metrics, models.Metric{
			ID:    key,
			MType: models.Counter,
			Delta: &value,
		})
	}
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metrics); err != nil {
		return err
	}
	return nil
}
