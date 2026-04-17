package config

import (
	"os"
	"strconv"
)

type AppConfig struct {
	HTTPPort    int
	DataDir     string
	LogLevel    string
	GatewayBind string
}

func Load() AppConfig {
	port := 3456
	if value := os.Getenv("HTTP_PORT"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			port = parsed
		}
	}

	return AppConfig{
		HTTPPort:    port,
		DataDir:     "./data",
		LogLevel:    "debug",
		GatewayBind: "127.0.0.1",
	}
}
