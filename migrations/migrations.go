package migrations

import (
	"context"
	"database/sql"
)

func Run(ctx context.Context, db *sql.DB) error
