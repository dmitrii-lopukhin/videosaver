package handlers

import (
	"context"
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/cache"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/jobs"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/normalize"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v3"
)

// Classification result for an inline URL.
type Classification int

const (
	ClassifyUnknown   Classification = iota
	ClassifyCacheHit                 // file_id cached → video goes directly into chat
	ClassifyCacheMiss                // no file_id yet → switch_pm to download first
)

// CacheClient is the subset of cache.Client used by InlineHandler.
type CacheClient interface {
	Get(ctx context.Context, key string) (string, error)
	GetJSON(ctx context.Context, key string, v any) (bool, error)
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

// JobEnqueuer is the subset of jobs.Queue used by InlineHandler.
type JobEnqueuer interface {
	Enqueue(ctx context.Context, job jobs.Job) (string, error)
}

// ExtractorRegistry routes URLs to extractors.
type ExtractorRegistry interface {
	For(url string) (extractors.Extractor, bool)
}

// InlineHandler handles Telegram inline queries for video URLs.
type InlineHandler struct {
	registry   ExtractorRegistry
	cache      CacheClient
	queue      JobEnqueuer
	timeoutSec int
}

func NewInline(registry ExtractorRegistry, cache CacheClient, queue JobEnqueuer, timeoutSec int) *InlineHandler {
	return &InlineHandler{registry: registry, cache: cache, queue: queue, timeoutSec: timeoutSec}
}

// Classify determines how to respond to an inline URL.
func (h *InlineHandler) Classify(url string) Classification {
	if _, ok := h.registry.For(url); !ok {
		return ClassifyUnknown
	}
	norm, err := normalize.Normalize(url)
	if err != nil {
		return ClassifyUnknown
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.timeoutSec)*time.Second)
	defer cancel()

	fileID, _ := h.cache.Get(ctx, cache.VideoFileIDKey(norm, false, "best"))
	if fileID != "" {
		return ClassifyCacheHit
	}
	return ClassifyCacheMiss
}

// Handle is the telebot inline query handler.
func (h *InlineHandler) Handle(log zerolog.Logger) tele.HandlerFunc {
	return func(c tele.Context) error {
		query := c.Query().Text
		if query == "" {
			return c.Answer(&tele.QueryResponse{})
		}

		if _, ok := h.registry.For(query); !ok {
			return c.Answer(&tele.QueryResponse{})
		}

		norm, err := normalize.Normalize(query)
		if err != nil {
			return c.Answer(&tele.QueryResponse{})
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.timeoutSec)*time.Second)
		defer cancel()

		// Check if we already have a Telegram file_id — send directly into chat.
		fileID, _ := h.cache.Get(ctx, cache.VideoFileIDKey(norm, false, "best"))
		if fileID != "" {
			log.Info().Str("url", norm).Msg("inline: cache hit, returning file_id")
			return c.Answer(&tele.QueryResponse{
				Results: tele.Results{
					&tele.VideoResult{
						Cache: fileID,
						Title: "Видео",
					},
				},
			})
		}

		// Cache miss: enqueue job and send user to PM to download.
		jobID, err := h.queue.Enqueue(ctx, jobs.Job{
			URL:     norm,
			UserID:  c.Query().Sender.ID,
			Quality: "best",
		})
		if err != nil {
			log.Error().Err(err).Msg("inline: enqueue job")
			return c.Answer(&tele.QueryResponse{})
		}

		log.Info().Str("url", norm).Str("job_id", jobID).Msg("inline: cache miss, job enqueued")
		return c.Answer(&tele.QueryResponse{
			SwitchPMText:      "Получить видео",
			SwitchPMParameter: jobID,
		})
	}
}
