package migrations

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var migrationFiles embed.FS

func Run(pool *sql.DB) error {
	sourceDriver, err := iofs.New(migrationFiles, ".")
	if err != nil {
		return err
	}
	databaseDriver, err := postgres.WithInstance(pool, &postgres.Config{})
	if err != nil {
		return err
	}
	migration, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"postgres",
		databaseDriver,
	)
	if err != nil {
		return err
	}
	if err = migration.Up(); errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}
