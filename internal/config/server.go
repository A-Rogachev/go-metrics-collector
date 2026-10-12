package config

import (
	"flag"
	"os"
)

type ServerConfig struct {
	ServerAddress string
}

func GetServerConfig() ServerConfig {
	cfg := ServerConfig{}

	flag.StringVar(&cfg.ServerAddress, "a", ":8080", "address and port to run server")
	flag.Parse()

	addr, exists := os.LookupEnv("ADDRESS")
	if exists && addr != "" {
		cfg.ServerAddress = addr
	}
	return cfg
}
