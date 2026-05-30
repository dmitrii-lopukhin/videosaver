package handlers_test

import (
	"context"
	"testing"
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/jobs"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/telegram/handlers"
)

type fakeRegistry struct{ canHandle bool }

func (f *fakeRegistry) For(url string) (extractors.Extractor, bool) {
	if f.canHandle {
		return &fakeExtractor{}, true
	}
	return nil, false
}

type fakeExtractor struct{}

func (f *fakeExtractor) CanHandle(_ string) bool { return true }
func (f *fakeExtractor) Resolve(_ context.Context, _ string, _ extractors.ResolveOpts) (*extractors.VideoResult, error) {
	return &extractors.VideoResult{DirectURL: "https://cdn/v.mp4", Title: "t", DurationSec: 10, SizeBytes: -1}, nil
}

type fakeCache struct{ hit bool }

func (f *fakeCache) GetJSON(_ context.Context, _ string, _ any) (bool, error) { return f.hit, nil }
func (f *fakeCache) Lock(_ context.Context, _ string, _ time.Duration) (bool, error) {
	return true, nil
}

type fakeQueue struct{ enqueuedURL string }

func (f *fakeQueue) Enqueue(_ context.Context, j jobs.Job) (string, error) {
	f.enqueuedURL = j.URL
	return "test-job-id", nil
}

func TestInlineHandler_UnknownURL(t *testing.T) {
	h := handlers.NewInline(&fakeRegistry{canHandle: false}, &fakeCache{}, &fakeQueue{}, 8)
	result := h.Classify("https://youtube.com/watch?v=x")
	if result != handlers.ClassifyUnknown {
		t.Errorf("expected ClassifyUnknown, got %v", result)
	}
}

func TestInlineHandler_CacheHit(t *testing.T) {
	h := handlers.NewInline(&fakeRegistry{canHandle: true}, &fakeCache{hit: true}, &fakeQueue{}, 8)
	result := h.Classify("https://instagram.com/p/ABC/")
	if result != handlers.ClassifyCacheHit {
		t.Errorf("expected ClassifyCacheHit, got %v", result)
	}
}

func TestInlineHandler_CacheMiss(t *testing.T) {
	h := handlers.NewInline(&fakeRegistry{canHandle: true}, &fakeCache{hit: false}, &fakeQueue{}, 8)
	result := h.Classify("https://instagram.com/p/ABC/")
	if result != handlers.ClassifyCacheMiss {
		t.Errorf("expected ClassifyCacheMiss, got %v", result)
	}
}
