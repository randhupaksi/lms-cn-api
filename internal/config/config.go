package config

import "os"

type Config struct {
	AppEnv    string
	Port      string
	APIPrefix string
	AppName   string
}

func Load() *Config {
	return &Config{
		AppEnv:    valueOrDefault("APP_ENV", "development"),
		Port:      valueOrDefault("PORT", "8080"),
		APIPrefix: valueOrDefault("API_PREFIX", "/api/v1"),
		AppName:   valueOrDefault("APP_NAME", "Ranvex API"),
	}
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
