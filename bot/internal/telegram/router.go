package telegram

import (
	"time"

	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v3"
)

func NewBot(token string, log zerolog.Logger) (*tele.Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	registerHandlers(b, log)
	return b, nil
}

func registerHandlers(b *tele.Bot, log zerolog.Logger) {
	b.Handle("/start", func(c tele.Context) error {
		log.Info().Int64("user_id", c.Sender().ID).Msg("/start received")
		return c.Send("VideoSaver MVP в разработке. Скоро всё будет.")
	})
}
