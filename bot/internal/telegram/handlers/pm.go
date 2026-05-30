package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/cache"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/download"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/jobs"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/normalize"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v3"
)

// JobDequeuer is the subset of jobs.Queue used by PMHandler.
type JobDequeuer interface {
	Dequeue(ctx context.Context, id string) (*jobs.Job, error)
	Delete(ctx context.Context, id string) error
}

// CacheSetter is the subset of cache.Client used by PMHandler.
type CacheSetter interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string) error
}

// PMHandler handles /start <job_id> messages in personal chat.
type PMHandler struct {
	registry         ExtractorRegistry
	queue            JobDequeuer
	cache            CacheSetter
	pmTimeoutSec     int
	downloadMaxBytes int64
	overrideExt      extractors.Extractor
}

func NewPM(registry ExtractorRegistry, queue JobDequeuer, cache CacheSetter, pmTimeoutSec int, downloadMaxBytes int64) *PMHandler {
	return &PMHandler{
		registry:         registry,
		queue:            queue,
		cache:            cache,
		pmTimeoutSec:     pmTimeoutSec,
		downloadMaxBytes: downloadMaxBytes,
	}
}

// OverrideExtractor replaces the registry lookup — for unit tests only.
func (h *PMHandler) OverrideExtractor(e extractors.Extractor) {
	h.overrideExt = e
}

// ProcessJob resolves the job and calls sendFn with the result.
// sendFn returns (fileID, error) — fileID is cached in Redis if non-empty.
func (h *PMHandler) ProcessJob(ctx context.Context, jobID string, userID int64, sendFn func(*extractors.VideoResult) (string, error)) error {
	job, err := h.queue.Dequeue(ctx, jobID)
	if err != nil {
		return err
	}

	ext := h.overrideExt
	if ext == nil {
		var ok bool
		ext, ok = h.registry.For(job.URL)
		if !ok {
			return fmt.Errorf("pm: no extractor for %s", job.URL)
		}
	}

	opts := extractors.ResolveOpts{Audio: job.Audio, Quality: job.Quality}
	result, err := ext.Resolve(ctx, job.URL, opts)
	if err != nil {
		return err
	}

	norm, _ := normalize.Normalize(job.URL)
	lockKey := cache.VideoLockKey(norm)
	cacheKey := cache.VideoKey(norm, job.Audio, job.Quality)

	if locked, _ := h.cache.Lock(ctx, lockKey, 30*time.Second); locked {
		defer h.cache.Unlock(ctx, lockKey)
		_ = h.cache.SetJSON(ctx, cacheKey, result, 24*time.Hour)
	}

	fileID, err := sendFn(result)
	if err != nil {
		return fmt.Errorf("pm: send: %w", err)
	}

	if fileID != "" {
		_ = h.cache.Set(ctx, cache.VideoFileIDKey(norm, job.Audio, job.Quality), fileID, 24*time.Hour)
	}

	_ = h.queue.Delete(ctx, jobID)
	return nil
}

// Handle returns the telebot handler for /start <job_id>.
func (h *PMHandler) Handle(bot *tele.Bot, log zerolog.Logger) tele.HandlerFunc {
	return func(c tele.Context) error {
		payload := c.Message().Payload
		if payload == "" {
			return c.Send("Привет! Отправь мне ссылку из инлайн-режима.")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.pmTimeoutSec)*time.Second)
		defer cancel()

		sendFn := func(result *extractors.VideoResult) (string, error) {
			return download.StreamToTelegram(
				ctx, bot, c.Recipient(),
				result.DirectURL, result.DurationSec,
				h.downloadMaxBytes,
			)
		}

		if err := h.ProcessJob(ctx, payload, c.Sender().ID, sendFn); err != nil {
			log.Error().Err(err).Str("job_id", payload).Msg("pm: process job failed")
			return c.Send("Не удалось получить видео. Попробуй ещё раз.")
		}
		return nil
	}
}
