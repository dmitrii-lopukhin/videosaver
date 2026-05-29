package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken         string
	RedisURL         string
	InstaResolverURL string
	CacheTTLSec      int
	LogLevel         string
}

func Load() (*Config, error) {
	_ = godotenv.Load() // .env is optional (docker-compose injects via env_file)

	cfg := &Config{
		BotToken:         os.Getenv("BOT_TOKEN"),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379/0"),
		InstaResolverURL: getEnv("INSTA_RESOLVER_URL", "http://insta-resolver:8000"),
		CacheTTLSec:      getEnvInt("CACHE_TTL_SEC", 86400),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
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
