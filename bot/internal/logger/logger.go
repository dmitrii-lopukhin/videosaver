package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func New(level string, out io.Writer) zerolog.Logger {
	if out == nil {
		out = os.Stdout
	}
	zerolog.TimeFieldFormat = time.RFC3339

	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	return zerolog.New(out).Level(lvl).With().Timestamp().Logger()
}
