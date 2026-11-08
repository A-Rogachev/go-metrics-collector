package service

import (
	"context"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

type MetricRepository interface {
	UpsertMetric(ctx context.Context, metric models.Metric) error
}

type DBRepository interface {
	UpsertMetric(ctx context.Context, metric models.Metric) error
	UpsertMetricBatch(ctx context.Context, metric []models.Metric) error
}

type MetricDBSaver struct {
	repository DBRepository
}

func NewMetricDBSaver(repository DBRepository) *MetricDBSaver {
	return &MetricDBSaver{
		repository: repository,
	}
}

func (s *MetricDBSaver) SaveOnUpdate(
	ctx context.Context,
	metric models.Metric,
) error {
	return s.repository.UpsertMetric(ctx, metric)
}

func (s *MetricDBSaver) SaveBatchOnUpdate(
	ctx context.Context,
	metrics []models.Metric,
) error {
	return s.repository.UpsertMetricBatch(ctx, metrics)
}
