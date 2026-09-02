package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	Port                   string
	APIPrefix              string
	AppName                string
	MigrationsPath         string
	DatabaseURL            string
	JWTSecret              string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	AllowedOrigins         []string
	CookieDomain           string
	CookieSecure           bool
	DatabaseMaxOpen        int
	DatabaseMaxIdle        int
	DatabaseMaxLife        time.Duration
	ShutdownTimeout        time.Duration
	LoginRateLimit         int
	LoginRateWindow        time.Duration
	LoginAccountRateLimit  int
	LoginAccountRateWindow time.Duration
	SeedDemoData           bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:                 valueOrDefault("APP_ENV", "development"),
		Port:                   valueOrDefault("PORT", "8080"),
		APIPrefix:              valueOrDefault("API_PREFIX", "/api/v1"),
		AppName:                valueOrDefault("APP_NAME", "Citra Negara LMS API"),
		MigrationsPath:         valueOrDefault("MIGRATIONS_PATH", "migrations"),
		DatabaseURL:            strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:              strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AccessTokenTTL:         durationOrDefault("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:        durationOrDefault("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		AllowedOrigins:         csvOrDefault("ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		CookieDomain:           strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")),
		CookieSecure:           boolOrDefault("COOKIE_SECURE", false),
		DatabaseMaxOpen:        intOrDefault("DATABASE_MAX_OPEN", 50),
		DatabaseMaxIdle:        intOrDefault("DATABASE_MAX_IDLE", 10),
		DatabaseMaxLife:        durationOrDefault("DATABASE_MAX_LIFETIME", 30*time.Minute),
		ShutdownTimeout:        durationOrDefault("SHUTDOWN_TIMEOUT", 10*time.Second),
		LoginRateLimit:         intOrDefault("LOGIN_RATE_LIMIT", 2500),
		LoginRateWindow:        durationOrDefault("LOGIN_RATE_WINDOW", time.Minute),
		LoginAccountRateLimit:  intOrDefault("LOGIN_ACCOUNT_RATE_LIMIT", 10),
		LoginAccountRateWindow: durationOrDefault("LOGIN_ACCOUNT_RATE_WINDOW", 5*time.Minute),
	}
	cfg.SeedDemoData = boolOrDefault("SEED_DEMO_DATA", cfg.AppEnv != "production")

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must contain at least 32 characters")
	}
	if c.AppEnv == "production" && c.SeedDemoData {
		return errors.New("SEED_DEMO_DATA must be false in production")
	}
	if !strings.HasPrefix(c.APIPrefix, "/") {
		return errors.New("API_PREFIX must start with a slash")
	}
	return nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func intOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func boolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func csvOrDefault(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
