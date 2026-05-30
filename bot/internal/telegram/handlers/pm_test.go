package handlers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/jobs"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/telegram/handlers"
)

type fakeJobQueue struct {
	job *jobs.Job
	err error
}

func (f *fakeJobQueue) Dequeue(_ context.Context, _ string) (*jobs.Job, error) {
	return f.job, f.err
}
func (f *fakeJobQueue) Delete(_ context.Context, _ string) error { return nil }

type fakeFullExtractor struct {
	result *extractors.VideoResult
	err    error
}

func (f *fakeFullExtractor) CanHandle(_ string) bool { return true }
func (f *fakeFullExtractor) Resolve(_ context.Context, _ string, _ extractors.ResolveOpts) (*extractors.VideoResult, error) {
	return f.result, f.err
}

type fakeSetter struct{}

func (f *fakeSetter) Set(_ context.Context, _, _ string, _ time.Duration) error          { return nil }
func (f *fakeSetter) SetJSON(_ context.Context, _ string, _ any, _ time.Duration) error  { return nil }
func (f *fakeSetter) Lock(_ context.Context, _ string, _ time.Duration) (bool, error)    { return true, nil }
func (f *fakeSetter) Unlock(_ context.Context, _ string) error                           { return nil }

func noSend(_ *extractors.VideoResult) (string, error) { return "", nil }

func TestPMHandler_JobNotFound(t *testing.T) {
	h := handlers.NewPM(
		&fakeRegistry{canHandle: true},
		&fakeJobQueue{err: jobs.ErrNotFound},
		&fakeSetter{},
		300, 52428800,
	)
	err := h.ProcessJob(context.Background(), "bad-id", 1, noSend)
	if !errors.Is(err, jobs.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPMHandler_ResolveError(t *testing.T) {
	h := handlers.NewPM(
		&fakeRegistry{canHandle: true},
		&fakeJobQueue{job: &jobs.Job{URL: "https://instagram.com/p/X/", Quality: "best"}},
		&fakeSetter{},
		300, 52428800,
	)
	h.OverrideExtractor(&fakeFullExtractor{err: extractors.ErrNotFound})
	err := h.ProcessJob(context.Background(), "id", 1, noSend)
	if !errors.Is(err, extractors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPMHandler_Success(t *testing.T) {
	vr := &extractors.VideoResult{DirectURL: "https://cdn/v.mp4", Title: "t", DurationSec: 5, SizeBytes: -1}
	h := handlers.NewPM(
		&fakeRegistry{canHandle: true},
		&fakeJobQueue{job: &jobs.Job{URL: "https://instagram.com/p/X/", Quality: "best"}},
		&fakeSetter{},
		300, 52428800,
	)
	h.OverrideExtractor(&fakeFullExtractor{result: vr})

	var sent *extractors.VideoResult
	err := h.ProcessJob(context.Background(), "id", 1, func(r *extractors.VideoResult) (string, error) {
		sent = r
		return "telegram-file-id-123", nil
	})
	if err != nil {
		t.Fatalf("ProcessJob: %v", err)
	}
	if sent == nil || sent.DirectURL != vr.DirectURL {
		t.Errorf("send callback not called with correct result")
	}
}
