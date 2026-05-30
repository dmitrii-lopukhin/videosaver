package logger

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var timeFormatOnce sync.Once

func New(level string, out io.Writer) zerolog.Logger {
	if out == nil {
		out = os.Stdout
	}
	timeFormatOnce.Do(func() {
		zerolog.TimeFieldFormat = time.RFC3339
	})

	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	return zerolog.New(out).Level(lvl).With().Timestamp().Logger()
}
