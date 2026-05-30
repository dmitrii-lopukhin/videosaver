package download

import (
	"context"
	"fmt"
	"io"

	tele "gopkg.in/telebot.v3"
)

// StreamToTelegram downloads the video at directURL and sends it to recip.
func StreamToTelegram(
	ctx context.Context,
	bot *tele.Bot,
	recip tele.Recipient,
	directURL string,
	title string,
	durationSec int,
	maxBytes int64,
) error {
	rc, err := Fetch(ctx, directURL, maxBytes)
	if err != nil {
		return fmt.Errorf("stream: fetch: %w", err)
	}
	defer rc.Close()

	video := &tele.Video{
		File:     tele.FromReader(rc),
		Caption:  title,
		Duration: durationSec,
	}

	_, err = bot.Send(recip, video)
	return err
}

// compile-time check
var _ io.ReadCloser = (*limitedReadCloser)(nil)
