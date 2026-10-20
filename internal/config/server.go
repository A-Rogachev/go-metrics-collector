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
	StoreInterval   int
	FileStoragePath string
	Restore         bool
}

type ServerConfig struct {
	ServerAddress string
	LogLevel      string
	StorageConfig FileStorageConfig
}

func GetServerConfig() ServerConfig {
	cfg := ServerConfig{}

	flag.StringVar(&cfg.ServerAddress, "a", ":8080", "address and port to run server")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "log level")
	flag.IntVar(&cfg.StorageConfig.StoreInterval, "i", 300, "how often save metrics into file")
	flag.StringVar(&cfg.StorageConfig.FileStoragePath, "f", "./temp.file", "filename path")
	flag.BoolVar(&cfg.StorageConfig.Restore, "r", true, "restore after server reload")
	flag.Parse()

	addr, exists := os.LookupEnv("ADDRESS")
	if exists && addr != "" {
		cfg.ServerAddress = addr
	}

	storageInterval, exists := os.LookupEnv("STORE_INTERVAL")
	if exists {
		val, err := strconv.Atoi(storageInterval)
		if err == nil && val <= MaxStorageInterval && val >= MinStorageInterval {
			cfg.StorageConfig.StoreInterval = val
		}
	}

	restore, exists := os.LookupEnv("RESTORE")
	if exists {
		val, err := strconv.ParseBool(restore)
		if err == nil {
			cfg.StorageConfig.Restore = val
		}
	}

	path, exists := os.LookupEnv("FILE_STORAGE_PATH")
	if exists {
		file, err := os.OpenFile(path, os.O_RDONLY, 0444)
		if err == nil {
			defer file.Close()
			cfg.StorageConfig.FileStoragePath = path
		}
	}

	return cfg
}
