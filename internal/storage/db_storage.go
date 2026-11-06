package storage

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

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
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		return err
	}
	return nil
}
