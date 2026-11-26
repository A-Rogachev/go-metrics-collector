package config

import (
	"flag"
	"os"
	"strconv"
)

const (
	MaxStorageInterval = 60 * 60 // 1 час
	MinStorageInterval = 0
)

type FileStorageConfig struct {
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

type ServerConfig struct {
	ServerAddress string `env:"ADDRESS"`
	LogLevel      string
	rawSecretKey  string `env:"KEY"`
	SecretKey     *string
	FileStorageConfig
}

type DatabaseConfig struct {
	DriverName string
	DSN        string
}

func GetServerConfigs() (ServerConfig, DatabaseConfig) {
	var serverCfg ServerConfig
	var databaseCfg DatabaseConfig

	flag.StringVar(&serverCfg.ServerAddress, "a", ":8080", "address and port to run server")
	flag.StringVar(&serverCfg.LogLevel, "log-level", "info", "log level")
	flag.IntVar(&serverCfg.StoreInterval, "i", 300, "how often save metrics into file")
	flag.StringVar(&serverCfg.FileStoragePath, "f", "./temp.file", "filename path")
	flag.BoolVar(&serverCfg.Restore, "r", true, "restore after server reload")
	flag.StringVar(&serverCfg.rawSecretKey, "k", "", "key for metrics hash check")

	flag.StringVar(&databaseCfg.DSN, "d", "", "database connection settings")
	flag.StringVar(&databaseCfg.DriverName, "dn", "pgx", "driver name")

	flag.Parse()

	addr, exists := os.LookupEnv("ADDRESS")
	if exists && addr != "" {
		serverCfg.ServerAddress = addr
	}

	storageInterval, exists := os.LookupEnv("STORE_INTERVAL")
	if exists {
		val, err := strconv.Atoi(storageInterval)
		if err == nil && val <= MaxStorageInterval && val >= MinStorageInterval {
			serverCfg.StoreInterval = val
		}
	}

	restore, exists := os.LookupEnv("RESTORE")
	if exists {
		val, err := strconv.ParseBool(restore)
		if err == nil {
			serverCfg.Restore = val
		}
	}

	path, exists := os.LookupEnv("FILE_STORAGE_PATH")
	if exists {
		serverCfg.FileStoragePath = path
	}

	key, exists := os.LookupEnv("KEY")
	if exists {
		serverCfg.rawSecretKey = key
	}
	if serverCfg.rawSecretKey != "" {
		serverCfg.SecretKey = &serverCfg.rawSecretKey
	}

	dsn, exists := os.LookupEnv("DATABASE_DSN")
	if exists {
		databaseCfg.DSN = dsn
	}

	return serverCfg, databaseCfg
}
