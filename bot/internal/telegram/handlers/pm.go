package handlers

import (
	"context"
	"fmt"
	"strings"
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
	Get(ctx context.Context, key string) (string, error)
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

// ProcessURL is the testable core of the DM "send me a link" feature.
// It resolves rawURL directly (no job queue): on a file_id cache hit it calls
// sendCached, otherwise it resolves via the registry and calls sendFresh,
// caching the resulting file_id. Returns handled=false when rawURL is not a
// supported video link, so the caller can reply with help text.
func (h *PMHandler) ProcessURL(
	ctx context.Context,
	rawURL string,
	sendCached func(fileID string) error,
	sendFresh func(*extractors.VideoResult) (string, error),
) (bool, error) {
	if _, ok := h.registry.For(rawURL); !ok {
		return false, nil
	}
	norm, err := normalize.Normalize(rawURL)
	if err != nil {
		return false, nil
	}

	const audio, quality = false, "best"

	// Cache hit: send the existing Telegram file_id, no download needed.
	if fileID, _ := h.cache.Get(ctx, cache.VideoFileIDKey(norm, audio, quality)); fileID != "" {
		return true, sendCached(fileID)
	}

	ext := h.overrideExt
	if ext == nil {
		var ok bool
		ext, ok = h.registry.For(norm)
		if !ok {
			return false, nil
		}
	}

	result, err := ext.Resolve(ctx, norm, extractors.ResolveOpts{Audio: audio, Quality: quality})
	if err != nil {
		return true, fmt.Errorf("pm: resolve %s: %w", norm, err)
	}

	fileID, err := sendFresh(result)
	if err != nil {
		return true, fmt.Errorf("pm: send: %w", err)
	}
	if fileID != "" {
		_ = h.cache.Set(ctx, cache.VideoFileIDKey(norm, audio, quality), fileID, 24*time.Hour)
	}
	return true, nil
}

// HandleText returns the telebot handler for plain text in a private chat:
// the user sends a video link and the bot downloads and returns the video.
func (h *PMHandler) HandleText(bot *tele.Bot, log zerolog.Logger) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Chat() == nil || c.Chat().Type != tele.ChatPrivate {
			return nil // only react to direct messages
		}
		text := strings.TrimSpace(c.Message().Text)
		if text == "" {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.pmTimeoutSec)*time.Second)
		defer cancel()

		sendCached := func(fileID string) error {
			return c.Send(&tele.Video{File: tele.File{FileID: fileID}})
		}
		sendFresh := func(result *extractors.VideoResult) (string, error) {
			return download.StreamToTelegram(
				ctx, bot, c.Recipient(),
				result.DirectURL, result.DurationSec,
				h.downloadMaxBytes,
			)
		}

		handled, err := h.ProcessURL(ctx, text, sendCached, sendFresh)
		if err != nil {
			log.Error().Err(err).Str("text", text).Msg("pm: process url failed")
			return c.Send("Не удалось получить видео. Попробуй ещё раз.")
		}
		if !handled {
			return c.Send("Пришли ссылку на видео (Instagram и т.п.), и я скачаю.")
		}
		return nil
	}
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
