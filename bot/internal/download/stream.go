package download

import (
	"context"
	"fmt"
	"io"

	tele "gopkg.in/telebot.v3"
)

// StreamToTelegram downloads the video at directURL and sends it to recip.
// Returns the Telegram file_id of the sent video (for caching in subsequent inline results).
func StreamToTelegram(
	ctx context.Context,
	bot *tele.Bot,
	recip tele.Recipient,
	directURL string,
	durationSec int,
	maxBytes int64,
) (string, error) {
	rc, err := Fetch(ctx, directURL, maxBytes)
	if err != nil {
		return "", fmt.Errorf("stream: fetch: %w", err)
	}
	defer rc.Close()

	video := &tele.Video{
		File:     tele.FromReader(rc),
		Duration: durationSec,
	}

	msg, err := bot.Send(recip, video)
	if err != nil {
		return "", err
	}
	if msg != nil && msg.Video != nil {
		return msg.Video.FileID, nil
	}
	return "", nil
}

// compile-time check
var _ io.ReadCloser = (*limitedReadCloser)(nil)
