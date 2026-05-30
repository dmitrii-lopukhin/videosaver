package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/cache"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/config"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors/insta"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/jobs"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/logger"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		println("config error:", err.Error())
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel, os.Stdout)
	log.Info().
		Str("insta_resolver_url", cfg.InstaResolverURL).
		Int("insta_resolver_timeout_sec", cfg.InstaResolverTimeoutSec).
		Int64("download_max_bytes", cfg.DownloadMaxBytes).
		Msg("videosaver bot starting")

	rdb, err := cache.New(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("redis init")
	}
	defer rdb.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := rdb.Ping(pingCtx); err != nil {
		log.Fatal().Err(err).Msg("redis ping")
	}
	log.Info().Msg("redis connected")

	instaExt := insta.New(cfg.InstaResolverURL, time.Duration(cfg.InstaResolverTimeoutSec)*time.Second)
	registry := extractors.NewRegistry(instaExt)

	opts, _ := redis.ParseURL(cfg.RedisURL)
	rawRDB := redis.NewClient(opts)
	jobQueue := jobs.NewQueue(rawRDB, 10*time.Minute)

	bot, err := telegram.NewBot(cfg.BotToken, telegram.Deps{
		Registry:         registry,
		Cache:            rdb,
		JobQueue:         jobQueue,
		InlineTimeoutSec: cfg.InlineTimeoutSec,
		PMTimeoutSec:     5 * 60,
		DownloadMaxBytes: cfg.DownloadMaxBytes,
		Log:              log,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("telegram init")
	}

	go bot.Start()
	log.Info().Msg("bot polling started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info().Msg("shutdown signal received")
	bot.Stop()
}
