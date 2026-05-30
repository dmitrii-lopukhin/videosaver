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
	ClassifyCacheHit                 // result already cached
	ClassifyCacheMiss                // job enqueued, show switch_pm
)

// CacheClient is the subset of cache.Client used by InlineHandler.
type CacheClient interface {
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

// Classify determines how to respond to an inline URL — used in unit tests and handler.
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

	var result extractors.VideoResult
	hit, _ := h.cache.GetJSON(ctx, cache.VideoKey(norm, false, "best"), &result)
	if hit {
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

		jobID, err := h.queue.Enqueue(ctx, jobs.Job{
			URL:     norm,
			UserID:  c.Query().Sender.ID,
			Quality: "best",
		})
		if err != nil {
			log.Error().Err(err).Msg("inline: enqueue job")
			return c.Answer(&tele.QueryResponse{})
		}

		log.Info().Str("url", norm).Str("job_id", jobID).Msg("inline: job enqueued")
		return c.Answer(&tele.QueryResponse{
			SwitchPMText:      "Получить видео",
			SwitchPMParameter: jobID,
		})
	}
}
