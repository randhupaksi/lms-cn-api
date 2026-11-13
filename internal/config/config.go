package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
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
	TrustedProxies         []string
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
	if err := validateEnvironmentValues(); err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:                 strings.ToLower(valueOrDefault("APP_ENV", "development")),
		Port:                   valueOrDefault("PORT", "8080"),
		APIPrefix:              valueOrDefault("API_PREFIX", "/api/v1"),
		AppName:                valueOrDefault("APP_NAME", "Citra Negara LMS API"),
		MigrationsPath:         valueOrDefault("MIGRATIONS_PATH", "migrations"),
		DatabaseURL:            strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:              strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AccessTokenTTL:         durationOrDefault("ACCESS_TOKEN_TTL", 24*time.Hour),
		RefreshTokenTTL:        durationOrDefault("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		AllowedOrigins:         csvOrDefault("ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		TrustedProxies:         csvOrEmpty("TRUSTED_PROXIES"),
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
	cfg.SeedDemoData = boolOrDefault("SEED_DEMO_DATA", cfg.AppEnv == "development")

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	switch c.AppEnv {
	case "development", "test", "staging", "production":
	default:
		return errors.New("APP_ENV must be development, test, staging, or production")
	}
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
	if c.RefreshTokenTTL <= c.AccessTokenTTL {
		return errors.New("REFRESH_TOKEN_TTL must be greater than ACCESS_TOKEN_TTL")
	}
	if c.DatabaseMaxIdle > c.DatabaseMaxOpen {
		return errors.New("DATABASE_MAX_IDLE must not exceed DATABASE_MAX_OPEN")
	}
	if !strings.HasPrefix(c.APIPrefix, "/") {
		return errors.New("API_PREFIX must start with a slash")
	}
	if c.Port != "" {
		port, err := strconv.Atoi(c.Port)
		if err != nil || port < 1 || port > 65535 {
			return errors.New("PORT must be an integer between 1 and 65535")
		}
	}
	for _, proxy := range c.TrustedProxies {
		if proxy == "0.0.0.0/0" || proxy == "::/0" {
			return errors.New("TRUSTED_PROXIES must not trust every address")
		}
		if strings.Contains(proxy, "/") {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				return errors.New("TRUSTED_PROXIES must contain valid IP addresses or CIDRs")
			}
		} else if net.ParseIP(proxy) == nil {
			return errors.New("TRUSTED_PROXIES must contain valid IP addresses or CIDRs")
		}
	}
	if c.AppEnv == "production" {
		if c.SeedDemoData {
			return errors.New("SEED_DEMO_DATA must be false in production")
		}
		if !c.CookieSecure {
			return errors.New("COOKIE_SECURE must be true in production")
		}
		if c.JWTSecret == "replace-with-a-long-random-secret" {
			return errors.New("JWT_SECRET must not use the example value in production")
		}
		for _, origin := range c.AllowedOrigins {
			parsed, err := url.ParseRequestURI(origin)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
				return fmt.Errorf("ALLOWED_ORIGINS must contain valid HTTPS origins in production: %q", origin)
			}
			hostname := strings.ToLower(parsed.Hostname())
			if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
				return fmt.Errorf("ALLOWED_ORIGINS must not contain local origins in production: %q", origin)
			}
		}
	} else if c.SeedDemoData && c.AppEnv != "development" {
		return errors.New("SEED_DEMO_DATA is only allowed when APP_ENV=development")
	}
	return nil
}

func valueOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func validateEnvironmentValues() error {
	for _, key := range []string{
		"ACCESS_TOKEN_TTL", "REFRESH_TOKEN_TTL", "DATABASE_MAX_LIFETIME",
		"SHUTDOWN_TIMEOUT", "LOGIN_RATE_WINDOW", "LOGIN_ACCOUNT_RATE_WINDOW",
	} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("%s must be a positive duration", key)
		}
	}
	for _, key := range []string{
		"DATABASE_MAX_OPEN", "DATABASE_MAX_IDLE", "LOGIN_RATE_LIMIT", "LOGIN_ACCOUNT_RATE_LIMIT",
	} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("%s must be a positive integer", key)
		}
	}
	for _, key := range []string{"COOKIE_SECURE", "SEED_DEMO_DATA"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("%s must be a boolean", key)
		}
	}
	return nil
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

func csvOrEmpty(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
