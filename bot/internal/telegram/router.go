package telegram

import (
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/jobs"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/telegram/handlers"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v3"
)

// CacheClient combines the cache interfaces required by all handlers.
// cache.Client satisfies all of these.
type CacheClient interface {
	handlers.CacheClient
	handlers.CacheSetter
	handlers.ChosenCacheClient
}

// Deps groups the dependencies the bot needs beyond the token.
type Deps struct {
	Registry         handlers.ExtractorRegistry
	Cache            CacheClient
	JobQueue         *jobs.Queue
	InlineTimeoutSec int
	PMTimeoutSec     int
	DownloadMaxBytes int64
	StorageChannelID int64
	Log              zerolog.Logger
}

func NewBot(token string, deps Deps) (*tele.Bot, error) {
	pref := tele.Settings{
		Token: token,
		Poller: &tele.LongPoller{
			Timeout: 10 * time.Second,
			AllowedUpdates: []string{
				"message",
				"inline_query",
				"chosen_inline_result",
				"callback_query",
			},
		},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	registerHandlers(b, deps)
	return b, nil
}

func registerHandlers(b *tele.Bot, deps Deps) {
	botUsername := b.Me.Username

	inlineH := handlers.NewInline(deps.Registry, deps.Cache, deps.JobQueue, deps.InlineTimeoutSec)
	pmH := handlers.NewPM(deps.Registry, deps.JobQueue, deps.Cache, deps.PMTimeoutSec, deps.DownloadMaxBytes)
	chosenH := handlers.NewChosen(deps.Registry, deps.JobQueue, deps.Cache, deps.StorageChannelID, deps.DownloadMaxBytes)

	b.Handle(tele.OnQuery, inlineH.Handle(deps.Log))
	b.Handle("/start", pmH.Handle(b, deps.Log))
	b.Handle(tele.OnInlineResult, chosenH.Handle(b, botUsername, deps.Log))
}
