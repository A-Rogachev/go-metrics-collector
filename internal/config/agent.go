package config

import "flag"

type APIConfig struct {
	Address string
}

type PollConfig struct {
	PollInterval int
}

type ReportConfig struct {
	ReportInterval int
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

	return cfg
}
