package telegram

import (
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/jobs"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/telegram/handlers"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v3"
)

// CacheClient combines the cache interfaces required by the handlers.
type CacheClient interface {
	handlers.CacheClient
	handlers.CacheSetter
}

// Deps groups the dependencies the bot needs beyond the token.
type Deps struct {
	Registry         handlers.ExtractorRegistry
	Cache            CacheClient
	JobQueue         *jobs.Queue
	InlineTimeoutSec int
	PMTimeoutSec     int
	DownloadMaxBytes int64
	Log              zerolog.Logger
}

func NewBot(token string, deps Deps) (*tele.Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	registerHandlers(b, deps)
	return b, nil
}

func registerHandlers(b *tele.Bot, deps Deps) {
	inlineH := handlers.NewInline(deps.Registry, deps.Cache, deps.JobQueue, deps.InlineTimeoutSec)
	pmH := handlers.NewPM(deps.Registry, deps.JobQueue, deps.Cache, deps.PMTimeoutSec, deps.DownloadMaxBytes)

	b.Handle(tele.OnQuery, inlineH.Handle(deps.Log))
	b.Handle("/start", pmH.Handle(b, deps.Log))
}
