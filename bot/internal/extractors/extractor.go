package extractors

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound = errors.New("extractor: not found")
	ErrBlocked  = errors.New("extractor: blocked or private")
	ErrTimeout  = errors.New("extractor: timeout")
)

type ErrRateLimited struct {
	RetryAfter time.Duration
}

func (e *ErrRateLimited) Error() string {
	return fmt.Sprintf("extractor: rate limited, retry after %s", e.RetryAfter)
}

type ResolveOpts struct {
	Audio   bool
	Quality string // "best" | "720" | "480"; empty means "best"
}

type VideoResult struct {
	DirectURL    string
	Title        string
	ThumbnailURL string
	DurationSec  int
	SizeBytes    int64 // -1 if unknown
	IsAudio      bool
}

type Extractor interface {
	CanHandle(url string) bool
	Resolve(ctx context.Context, url string, opts ResolveOpts) (*VideoResult, error)
}
