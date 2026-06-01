package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken                string
	RedisURL                string
	InstaResolverURL        string
	InstaResolverTimeoutSec int
	InlineTimeoutSec        int
	CacheTTLSec             int
	DownloadMaxBytes        int64
	StorageChannelID        int64
	LogLevel                string
}

func Load() (*Config, error) {
	_ = godotenv.Load() // .env is optional (docker-compose injects via env_file)

	cfg := &Config{
		BotToken:                os.Getenv("BOT_TOKEN"),
		RedisURL:                getEnv("REDIS_URL", "redis://localhost:6379/0"),
		InstaResolverURL:        getEnv("INSTA_RESOLVER_URL", "http://insta-resolver:8000"),
		InstaResolverTimeoutSec: getEnvInt("INSTA_RESOLVER_TIMEOUT_SEC", 30),
		InlineTimeoutSec:        getEnvInt("INLINE_TIMEOUT_SEC", 8),
		CacheTTLSec:             getEnvInt("CACHE_TTL_SEC", 86400),
		DownloadMaxBytes:        getEnvInt64("DOWNLOAD_MAX_BYTES", 52428800),
		StorageChannelID:        getEnvInt64("STORAGE_CHANNEL_ID", 0),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}
