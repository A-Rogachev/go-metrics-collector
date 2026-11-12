package repository

import (
	"context"
	"database/sql"

	models "github.com/A-Rogachev/go-metrics-collector/internal/model"
)

const upsertMetricQuery = `
    INSERT INTO metrics (id, type, delta, value)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT (id) DO UPDATE
    SET
        type = EXCLUDED.type,
        delta = metrics.delta + EXCLUDED.delta,
        value = EXCLUDED.value,
        updated_at = CURRENT_TIMESTAMP;
`

type PostgresMetricsRepository struct {
	db *sql.DB
}

func NewPostgresMetricsRepository(
	db *sql.DB,
) *PostgresMetricsRepository {
	return &PostgresMetricsRepository{
		db: db,
	}
}

func (r *PostgresMetricsRepository) UpsertMetric(
	ctx context.Context,
	metric models.Metric,
) error {
	_, err := r.db.ExecContext(
		ctx,
		upsertMetricQuery,
		metric.ID,
		metric.MType,
		metric.Delta,
		metric.Value,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *PostgresMetricsRepository) UpsertMetricBatch(
	ctx context.Context,
	metrics []models.Metric,
) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	for _, m := range metrics {
		_, err := tx.ExecContext(
			ctx,
			upsertMetricQuery,
			m.ID,
			m.MType,
			m.Delta,
			m.Value,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
