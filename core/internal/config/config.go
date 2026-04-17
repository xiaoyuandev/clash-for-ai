package config

type AppConfig struct {
	HTTPPort    int
	DataDir     string
	LogLevel    string
	GatewayBind string
}

func Load() AppConfig {
	return AppConfig{
		HTTPPort:    3456,
		DataDir:     "./data",
		LogLevel:    "debug",
		GatewayBind: "127.0.0.1",
	}
}
