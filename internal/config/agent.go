package config

import (
	"flag"
	"os"
	"strconv"
)

type APIConfig struct {
	Address string `env:"ADDRESS"`
}

type PollConfig struct {
	PollInterval int `env:"POLL_INTERVAL"`
}

type ReportConfig struct {
	ReportInterval int `env:"REPORT_INTERVAL"`
}

type Additional struct {
	LogLevel string
}

type AgentConfig struct {
	APIConfig
	PollConfig
	ReportConfig
	Additional
}

func GetAgentConfig() AgentConfig {
	cfg := AgentConfig{}

	flag.StringVar(&cfg.Additional.LogLevel, "log-level", "info", "log level")
	flag.StringVar(&cfg.APIConfig.Address, "a", "localhost:8080", "api address and port")
	flag.IntVar(&cfg.ReportConfig.ReportInterval, "r", 10, "how often send metrics via api")
	flag.IntVar(&cfg.PollConfig.PollInterval, "p", 2, "how often get metrics via runtime package")
	flag.Parse()

	address, exists := os.LookupEnv("ADDRESS")
	if exists && address != "" {
		cfg.Address = address
	}
	reportInterval, exists := os.LookupEnv("REPORT_INTERVAL")
	if exists {
		val, err := strconv.Atoi(reportInterval)
		if err == nil && val > 0 {
			cfg.ReportInterval = val
		}
	}
	pollInterval, exists := os.LookupEnv("POLL_INTERVAL")
	if exists {
		val, err := strconv.Atoi(pollInterval)
		if err == nil && val > 0 {
			cfg.PollInterval = val
		}
	}

	return cfg
}
