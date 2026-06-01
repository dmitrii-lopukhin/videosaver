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
	SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

// resolveMetaTTL bounds reuse of a resolved direct URL. Kept short because
// direct CDN URLs are signed and expire; it only avoids re-hitting the
// (rate-limited) resolver on every keystroke of the same inline query.
const resolveMetaTTL = 90 * time.Second

// directResolveTimeout is the budget for the synchronous resolve used when we
// must deliver the video by URL (non-private chats). It is deliberately larger
// than the inline timeout because the resolver (instaloader) can take 10-15s,
// and here there is no placeholder to fall back on.
const directResolveTimeout = 20 * time.Second

// chatTypePrivate is the only inline-query chat_type where the placeholder +
// chosen_inline_result + edit flow works reliably. Elsewhere Telegram either
// doesn't deliver chosen_inline_result (channel comment groups) or provides no
// editable inline_message_id ("Saved Messages", chat_type "sender"), so the
// placeholder could never become the video. In those contexts we resolve and
// send the video directly by URL instead.
const chatTypePrivate = "private"

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

		// Outside private chats the placeholder+edit flow below can't work
		// (no chosen_inline_result in channel comments; no editable
		// inline_message_id in Saved Messages). Resolve the direct URL now —
		// with its own longer budget — and let Telegram fetch the video itself.
		// There is no placeholder fallback here: it could never be replaced.
		if c.Query().ChatType != chatTypePrivate {
			rctx, rcancel := context.WithTimeout(context.Background(), directResolveTimeout)
			defer rcancel()

			vr, err := h.resolveCached(rctx, norm)
			if err != nil || vr.DirectURL == "" {
				log.Warn().Err(err).Str("url", norm).Str("chat_type", c.Query().ChatType).Msg("inline: direct-url resolve failed")
				return c.Answer(&tele.QueryResponse{})
			}
			log.Info().Str("url", norm).Str("chat_type", c.Query().ChatType).Msg("inline: returning direct url")
			return c.Answer(&tele.QueryResponse{
				Results: tele.Results{
					&tele.VideoResult{
						URL:      vr.DirectURL,
						MIME:     "video/mp4",
						ThumbURL: vr.ThumbnailURL,
						Title:    "Видео",
						Duration: vr.DurationSec,
					},
				},
				CacheTime: 10,
			})
		}

		// Cache miss: enqueue job, return a placeholder the user can tap.
		// chosen_inline_result fires when they do — ChosenHandler takes it from there.
		jobID, err := h.queue.Enqueue(ctx, jobs.Job{
			URL:     norm,
			UserID:  c.Query().Sender.ID,
			Quality: "best",
		})
		if err != nil {
			log.Error().Err(err).Msg("inline: enqueue job")
			return c.Answer(&tele.QueryResponse{})
		}

		log.Info().Str("url", norm).Str("job_id", jobID).Str("chat_type", c.Query().ChatType).Msg("inline: cache miss, placeholder sent")
		// A non-empty inline keyboard is required for Telegram to include
		// inline_message_id in chosen_inline_result, without which the
		// placeholder could never be edited into the video. The button doubles
		// as a fallback link and is dropped when the message becomes a video.
		return c.Answer(&tele.QueryResponse{
			Results: tele.Results{
				&tele.ArticleResult{
					ResultBase: tele.ResultBase{
						ID: jobID,
						ReplyMarkup: &tele.ReplyMarkup{
							InlineKeyboard: [][]tele.InlineButton{{
								{Text: "⏳ Загружаю видео…", URL: norm},
							}},
						},
					},
					Title: "⏳ Загружаю видео…",
					Text:  "⏳ Загружаю видео…",
				},
			},
		})
	}
}

// resolveCached returns resolved video metadata for norm, reusing a
// short-lived Redis cache so the (rate-limited) resolver isn't hit on every
// keystroke of the same inline query.
func (h *InlineHandler) resolveCached(ctx context.Context, norm string) (extractors.VideoResult, error) {
	var vr extractors.VideoResult
	if found, _ := h.cache.GetJSON(ctx, cache.VideoKey(norm, false, "best"), &vr); found && vr.DirectURL != "" {
		return vr, nil
	}
	ext, ok := h.registry.For(norm)
	if !ok {
		return extractors.VideoResult{}, extractors.ErrNotFound
	}
	res, err := ext.Resolve(ctx, norm, extractors.ResolveOpts{Quality: "best"})
	if err != nil {
		return extractors.VideoResult{}, err
	}
	_ = h.cache.SetJSON(ctx, cache.VideoKey(norm, false, "best"), res, resolveMetaTTL)
	return *res, nil
}
