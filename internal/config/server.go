package config

import "flag"

type ServerConfig struct {
	ServerAddress string
}

func GetServerConfig() ServerConfig {
	cfg := ServerConfig{}

	flag.StringVar(&cfg.ServerAddress, "a", ":8080", "address and port to run server")
	flag.Parse()

	return cfg
}
