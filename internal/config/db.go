package config

import (
	"flag"
	"os"
)

type DatabaseConfig struct {
	DriverName string
	DSN        string
}

func GetDatabaseConfig() DatabaseConfig {
	cfg := DatabaseConfig{}
	flag.StringVar(&cfg.DSN, "d", "", "database connection settings")
	flag.StringVar(&cfg.DriverName, "dn", "pgx", "driver name")
	flag.Parse()

	dsn, exists := os.LookupEnv("DATABASE_DSN")
	if exists {
		cfg.DSN = dsn
	}

	return cfg
}
