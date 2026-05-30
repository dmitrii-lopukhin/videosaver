package extractors_test

import (
	"testing"
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
)

func TestSentinelErrorsAreDistinct(t *testing.T) {
	errs := []error{extractors.ErrNotFound, extractors.ErrBlocked, extractors.ErrTimeout}
	for i, a := range errs {
		for j, b := range errs {
			if i != j && a == b {
				t.Errorf("sentinel errors %d and %d are the same value", i, j)
			}
		}
	}
}

func TestErrRateLimitedImplementsError(t *testing.T) {
	e := &extractors.ErrRateLimited{RetryAfter: 30 * time.Second}
	if e.Error() == "" {
		t.Error("ErrRateLimited.Error() returned empty string")
	}
}

func TestResolveOptsDefaults(t *testing.T) {
	var opts extractors.ResolveOpts
	if opts.Audio != false {
		t.Error("Audio should default to false")
	}
	if opts.Quality != "" {
		t.Error("Quality should default to empty string")
	}
}
