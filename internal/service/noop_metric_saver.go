package service

import (
	"context"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

type NoopMetricSaver struct{}

func NewNoopMetricSaver() *NoopMetricSaver {
	return &NoopMetricSaver{}
}

func (s *NoopMetricSaver) SaveOnUpdate(context.Context, models.Metric) error {
	return nil
}

func (s *NoopMetricSaver) SaveBatchOnUpdate(
	ctx context.Context,
	metrics []models.Metric,
) error {
	return nil
}
