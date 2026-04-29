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

	dataDir := "./data"
	if value := os.Getenv("LOCAL_GATEWAY_RUNTIME_DATA_DIR"); value != "" {
		dataDir = value
	} else if value := os.Getenv("CORE_DATA_DIR"); value != "" {
		dataDir = value
	}

	return AppConfig{
		HTTPPort:         port,
		DataDir:          dataDir,
		LogLevel:         "debug",
		GatewayBind:      "127.0.0.1",
		LogRetentionDays: logRetentionDays,
		LogMaxRecords:    logMaxRecords,
	}
}
