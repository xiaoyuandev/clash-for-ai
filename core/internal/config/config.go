package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type AppConfig struct {
	HTTPPort                      int
	DataDir                       string
	LogLevel                      string
	GatewayBind                   string
	WebAssetsDir                  string
	LogRetentionDays              int
	LogMaxRecords                 int
	LocalGatewayRuntimeKind       string
	LocalGatewayRuntimeExecutable string
	LocalGatewayRuntimeHost       string
	LocalGatewayRuntimePort       int
	LocalGatewayRuntimeDataDir    string
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
	if value := os.Getenv("CORE_DATA_DIR"); value != "" {
		dataDir = value
	}

	localGatewayPort := 3457
	if value := os.Getenv("LOCAL_GATEWAY_RUNTIME_PORT"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			localGatewayPort = parsed
		}
	}

	localGatewayHost := "127.0.0.1"
	if value := os.Getenv("LOCAL_GATEWAY_RUNTIME_HOST"); value != "" {
		localGatewayHost = value
	}

	localGatewayDataDir := filepath.Join(dataDir, "local-gateway")
	if value := os.Getenv("LOCAL_GATEWAY_RUNTIME_DATA_DIR"); value != "" {
		localGatewayDataDir = value
	}

	return AppConfig{
		HTTPPort:                      port,
		DataDir:                       dataDir,
		LogLevel:                      "debug",
		GatewayBind:                   "127.0.0.1",
		WebAssetsDir:                  os.Getenv("WEB_ASSETS_DIR"),
		LogRetentionDays:              logRetentionDays,
		LogMaxRecords:                 logMaxRecords,
		LocalGatewayRuntimeKind:       envOrDefault("LOCAL_GATEWAY_RUNTIME_KIND", "ai-mini-gateway"),
		LocalGatewayRuntimeExecutable: os.Getenv("LOCAL_GATEWAY_RUNTIME_EXECUTABLE"),
		LocalGatewayRuntimeHost:       localGatewayHost,
		LocalGatewayRuntimePort:       localGatewayPort,
		LocalGatewayRuntimeDataDir:    localGatewayDataDir,
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
