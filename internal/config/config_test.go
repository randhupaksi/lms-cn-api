package config

import (
	"strings"
	"testing"
	"time"
)

func validProductionConfig() *Config {
	return &Config{
		AppEnv:          "production",
		APIPrefix:       "/api/v1",
		DatabaseURL:     "user:password@tcp(database:3306)/lms_cn",
		JWTSecret:       strings.Repeat("a", 48),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		AllowedOrigins:  []string{"https://lms.citranegara.sch.id"},
		CookieSecure:    true,
		DatabaseMaxOpen: 50,
		DatabaseMaxIdle: 10,
		SeedDemoData:    false,
	}
}

func TestValidateAcceptsSecureProductionConfiguration(t *testing.T) {
	if err := validProductionConfig().validate(); err != nil {
		t.Fatalf("expected production configuration to be valid, got %v", err)
	}
}

func TestValidateRejectsUnsafeProductionConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "insecure cookie", mutate: func(cfg *Config) { cfg.CookieSecure = false }},
		{name: "example jwt secret", mutate: func(cfg *Config) { cfg.JWTSecret = "replace-with-a-long-random-secret" }},
		{name: "http origin", mutate: func(cfg *Config) { cfg.AllowedOrigins = []string{"http://lms.citranegara.sch.id"} }},
		{name: "local origin", mutate: func(cfg *Config) { cfg.AllowedOrigins = []string{"https://localhost:3000"} }},
		{name: "wildcard origin", mutate: func(cfg *Config) { cfg.AllowedOrigins = []string{"*"} }},
		{name: "demo seed", mutate: func(cfg *Config) { cfg.SeedDemoData = true }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validProductionConfig()
			test.mutate(cfg)
			if err := cfg.validate(); err == nil {
				t.Fatal("expected unsafe production configuration to be rejected")
			}
		})
	}
}

func TestValidateRejectsInvalidLifetimeAndPoolOrdering(t *testing.T) {
	t.Run("refresh token lifetime", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.RefreshTokenTTL = cfg.AccessTokenTTL
		if err := cfg.validate(); err == nil {
			t.Fatal("expected refresh token lifetime ordering to be rejected")
		}
	})

	t.Run("database pool", func(t *testing.T) {
		cfg := validProductionConfig()
		cfg.DatabaseMaxIdle = cfg.DatabaseMaxOpen + 1
		if err := cfg.validate(); err == nil {
			t.Fatal("expected invalid database pool ordering to be rejected")
		}
	})
}
