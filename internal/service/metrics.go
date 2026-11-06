package service

import (
	"fmt"
	"sync"
	"time"

	loggers "github.com/A-Rogachev/go-metrics-collector/internal/logger"
	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

type SnapshotProvider interface {
	Snapshot() models.MetricSnapshot
	LoadFromSnapshot(models.MetricSnapshot)
}

type SnapshotSaver interface {
	UpdateFromSnapshot(models.MetricSnapshot) error
	LoadSnapshotFromFile() (models.MetricSnapshot, error)
}

type MetricSaver struct {
	provider      SnapshotProvider
	saver         SnapshotSaver
	storeInterval time.Duration
	isPeriodical  bool
	mutex         sync.Mutex
	logger        loggers.Logger
}

func NewMetricSaver(
	provider SnapshotProvider,
	saver SnapshotSaver,
	interval time.Duration,
	logger loggers.Logger,
) *MetricSaver {
	return &MetricSaver{
		provider:      provider,
		saver:         saver,
		storeInterval: interval,
		isPeriodical:  interval != 0,
		logger:        logger,
	}
}

func (service *MetricSaver) saveNow() error {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	snapshot := service.provider.Snapshot()
	return service.saver.UpdateFromSnapshot(snapshot)
}

func (service *MetricSaver) SaveOnUpdate() error {
	if service.isPeriodical {
		return nil
	}
	return service.saveNow()
}

func (service *MetricSaver) RestoreMetrics() error {
	snapshot, err := service.saver.LoadSnapshotFromFile()
	if err != nil {
		return err
	}
	service.provider.LoadFromSnapshot(snapshot)
	return nil
}

func (service *MetricSaver) PeriodicalSave() {
	if !service.isPeriodical {
		return
	}
	for {
		time.Sleep(service.storeInterval)
		err := service.saveNow()
		if err != nil {
			service.logger.Error().Msg(fmt.Sprintf("Periodical save failed: %s", err))
		}
	}

}
