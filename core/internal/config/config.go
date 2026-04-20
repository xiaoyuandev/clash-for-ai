package config

import (
	"os"
	"strconv"
)

type AppConfig struct {
	HTTPPort         int
	DataDir          string
	LogLevel         string
	GatewayBind      string
	LogRetentionDays int
	LogMaxRecords    int
}

func Load() AppConfig {
	port := 3456
	if value := os.Getenv("HTTP_PORT"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			port = parsed
		}
	}

	logRetentionDays := 30
	if value := os.Getenv("LOG_RETENTION_DAYS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			logRetentionDays = parsed
		}
	}

	logMaxRecords := 10000
	if value := os.Getenv("LOG_MAX_RECORDS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			logMaxRecords = parsed
		}
	}

	return AppConfig{
		HTTPPort:         port,
		DataDir:          "./data",
		LogLevel:         "debug",
		GatewayBind:      "127.0.0.1",
		LogRetentionDays: logRetentionDays,
		LogMaxRecords:    logMaxRecords,
	}
}
