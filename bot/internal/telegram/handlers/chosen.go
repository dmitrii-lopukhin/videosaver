package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/cache"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/download"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/normalize"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v3"
)

// ChosenCacheClient stores file_ids after upload.
type ChosenCacheClient interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

// ChosenHandler handles chosen_inline_result — fires when the user taps the placeholder.
type ChosenHandler struct {
	registry         ExtractorRegistry
	queue            JobDequeuer
	cache            ChosenCacheClient
	storageChannelID int64
	downloadMaxBytes int64
	overrideExt      extractors.Extractor
}

func NewChosen(
	registry ExtractorRegistry,
	queue JobDequeuer,
	cache ChosenCacheClient,
	storageChannelID int64,
	downloadMaxBytes int64,
) *ChosenHandler {
	return &ChosenHandler{
		registry:         registry,
		queue:            queue,
		cache:            cache,
		storageChannelID: storageChannelID,
		downloadMaxBytes: downloadMaxBytes,
	}
}

// OverrideExtractor replaces registry lookup — for unit tests only.
func (h *ChosenHandler) OverrideExtractor(e extractors.Extractor) { h.overrideExt = e }

// ProcessChosen is the testable core: resolves the job and calls upload, edit, or fallback.
func (h *ChosenHandler) ProcessChosen(
	ctx context.Context,
	jobID string,
	upload func(result *extractors.VideoResult, maxBytes int64) (string, error),
	edit func(fileID string) error,
	fallback func(botUsername, jobID string) error,
) error {
	job, err := h.queue.Dequeue(ctx, jobID)
	if err != nil {
		return err
	}

	ext := h.overrideExt
	if ext == nil {
		var ok bool
		ext, ok = h.registry.For(job.URL)
		if !ok {
			_ = fallback("", jobID)
			return nil
		}
	}

	opts := extractors.ResolveOpts{Audio: job.Audio, Quality: job.Quality}
	result, err := ext.Resolve(ctx, job.URL, opts)
	if err != nil {
		_ = fallback("", jobID)
		return nil
	}

	fileID, err := upload(result, h.downloadMaxBytes)
	if err != nil {
		_ = fallback("", jobID)
		return nil
	}

	if err := edit(fileID); err != nil {
		return fmt.Errorf("chosen: edit inline message: %w", err)
	}

	norm, _ := normalize.Normalize(job.URL)
	_ = h.cache.Set(ctx, cache.VideoFileIDKey(norm, job.Audio, job.Quality), fileID, 24*time.Hour)
	_ = h.queue.Delete(ctx, jobID)
	return nil
}

// Handle returns the telebot handler for OnChosenInlineResult.
func (h *ChosenHandler) Handle(bot *tele.Bot, botUsername string, log zerolog.Logger) tele.HandlerFunc {
	return func(c tele.Context) error {
		ir := c.InlineResult()
		jobID := ir.ResultID
		if jobID == "" {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*60*time.Second)
		defer cancel()

		upload := func(result *extractors.VideoResult, maxBytes int64) (string, error) {
			return uploadToStorage(ctx, bot, h.storageChannelID, result, maxBytes)
		}

		edit := func(fileID string) error {
			_, err := bot.Edit(ir, &tele.Video{File: tele.File{FileID: fileID}})
			return err
		}

		fallback := func(_ string, jid string) error {
			deepLink := fmt.Sprintf("https://t.me/%s?start=%s", botUsername, jid)
			_, err := bot.Edit(ir,
				fmt.Sprintf("❌ Не получилось загрузить\\.\n[Получить в личке](%s)", deepLink),
				tele.ModeMarkdownV2)
			return err
		}

		if err := h.ProcessChosen(ctx, jobID, upload, edit, fallback); err != nil {
			log.Error().Err(err).Str("job_id", jobID).Msg("chosen: process failed")
		}
		return nil
	}
}

// uploadToStorage downloads directURL and sends to the storage channel, returning the file_id.
func uploadToStorage(ctx context.Context, bot *tele.Bot, channelID int64, result *extractors.VideoResult, maxBytes int64) (string, error) {
	rc, err := download.Fetch(ctx, result.DirectURL, maxBytes)
	if err != nil {
		return "", fmt.Errorf("chosen: fetch: %w", err)
	}
	defer rc.Close()

	msg, err := bot.Send(tele.ChatID(channelID), &tele.Video{
		File:     tele.FromReader(rc),
		Duration: result.DurationSec,
	})
	if err != nil {
		return "", fmt.Errorf("chosen: upload to storage channel: %w", err)
	}
	if msg.Video == nil {
		return "", fmt.Errorf("chosen: storage channel did not return a video message")
	}
	return msg.Video.FileID, nil
}
