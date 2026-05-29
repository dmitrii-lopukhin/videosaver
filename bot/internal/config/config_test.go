package config

import (
	"os"
	"testing"
)

func TestLoad_AllFieldsFromEnv(t *testing.T) {
	t.Setenv("BOT_TOKEN", "test-token-123")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("INSTA_RESOLVER_URL", "http://insta-resolver:8000")
	t.Setenv("CACHE_TTL_SEC", "3600")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.BotToken != "test-token-123" {
		t.Errorf("BotToken = %q, want %q", cfg.BotToken, "test-token-123")
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if cfg.InstaResolverURL != "http://insta-resolver:8000" {
		t.Errorf("InstaResolverURL = %q", cfg.InstaResolverURL)
	}
	if cfg.CacheTTLSec != 3600 {
		t.Errorf("CacheTTLSec = %d", cfg.CacheTTLSec)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
}

func TestLoad_MissingBotTokenIsError(t *testing.T) {
	os.Unsetenv("BOT_TOKEN")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when BOT_TOKEN missing, got nil")
	}
}

func TestLoad_DefaultsWhenEnvAbsent(t *testing.T) {
	t.Setenv("BOT_TOKEN", "x")
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("CACHE_TTL_SEC")
	os.Unsetenv("LOG_LEVEL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("default RedisURL = %q", cfg.RedisURL)
	}
	if cfg.CacheTTLSec != 86400 {
		t.Errorf("default CacheTTLSec = %d, want 86400", cfg.CacheTTLSec)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("default LogLevel = %q", cfg.LogLevel)
	}
}
