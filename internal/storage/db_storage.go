package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/A-Rogachev/go-metrics-collector/internal/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type UnavailableDBHealthChecker struct{}

func (UnavailableDBHealthChecker) Ping(context.Context) error {
	return errors.New("database is not configured")
}

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(driverName string, dsn string) (*DBStorage, error) {
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	storage := DBStorage{db: db}
	return &storage, nil
}

func (s DBStorage) Ping(ctx context.Context) error {
	return repository.Retry(ctx, func() error {
		pingCtx, cancel := context.WithTimeout(
			ctx,
			2*time.Second,
		)
		defer cancel()

		return s.db.PingContext(pingCtx)
	})
}

func (s DBStorage) GetPool() *sql.DB {
	return s.db
}
