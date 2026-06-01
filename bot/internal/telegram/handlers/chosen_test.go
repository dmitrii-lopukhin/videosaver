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

type fakeChosenQueue struct {
	job *jobs.Job
	err error
}

func (f *fakeChosenQueue) Dequeue(_ context.Context, _ string) (*jobs.Job, error) {
	return f.job, f.err
}
func (f *fakeChosenQueue) Delete(_ context.Context, _ string) error { return nil }

type fakeChosenCache struct{ stored map[string]string }

func (f *fakeChosenCache) Set(_ context.Context, key, val string, _ time.Duration) error {
	if f.stored == nil {
		f.stored = map[string]string{}
	}
	f.stored[key] = val
	return nil
}

func TestChosenHandler_JobNotFound(t *testing.T) {
	h := handlers.NewChosen(
		&fakeRegistry{canHandle: true},
		&fakeChosenQueue{err: jobs.ErrNotFound},
		&fakeChosenCache{},
		-1001234567890,
		52428800,
	)

	editCalled := false
	err := h.ProcessChosen(context.Background(), "missing-job-id",
		func(_ *extractors.VideoResult, _ int64) (string, error) { return "", nil },
		func(_ string) error { editCalled = true; return nil },
		func(_, _ string) error { return nil },
	)
	if !errors.Is(err, jobs.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if editCalled {
		t.Error("edit should not be called when job not found")
	}
}

func TestChosenHandler_UploadSuccess_EditsWithFileID(t *testing.T) {
	h := handlers.NewChosen(
		&fakeRegistry{canHandle: true},
		&fakeChosenQueue{job: &jobs.Job{URL: "https://instagram.com/p/X/", Quality: "best"}},
		&fakeChosenCache{},
		-1001234567890,
		52428800,
	)
	h.OverrideExtractor(&fakeFullExtractor{
		result: &extractors.VideoResult{DirectURL: "https://cdn/v.mp4", DurationSec: 5, SizeBytes: -1},
	})

	var editedFileID string
	err := h.ProcessChosen(context.Background(), "job-id",
		func(_ *extractors.VideoResult, _ int64) (string, error) {
			return "tg-file-id-uploaded", nil
		},
		func(fileID string) error {
			editedFileID = fileID
			return nil
		},
		func(_, _ string) error { return errors.New("should not call fallback") },
	)
	if err != nil {
		t.Fatalf("ProcessChosen: %v", err)
	}
	if editedFileID != "tg-file-id-uploaded" {
		t.Errorf("edit called with %q, want %q", editedFileID, "tg-file-id-uploaded")
	}
}

func TestChosenHandler_UploadError_CallsFallback(t *testing.T) {
	h := handlers.NewChosen(
		&fakeRegistry{canHandle: true},
		&fakeChosenQueue{job: &jobs.Job{URL: "https://instagram.com/p/X/", Quality: "best"}},
		&fakeChosenCache{},
		-1001234567890,
		52428800,
	)
	h.OverrideExtractor(&fakeFullExtractor{
		result: &extractors.VideoResult{DirectURL: "https://cdn/v.mp4", DurationSec: 5, SizeBytes: -1},
	})

	fallbackCalled := false
	err := h.ProcessChosen(context.Background(), "job-id",
		func(_ *extractors.VideoResult, _ int64) (string, error) {
			return "", errors.New("upload failed")
		},
		func(_ string) error { return errors.New("should not edit on upload failure") },
		func(_, _ string) error {
			fallbackCalled = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fallbackCalled {
		t.Error("fallback should be called when upload fails")
	}
}

func TestChosenHandler_FileIDCachedAfterUpload(t *testing.T) {
	c := &fakeChosenCache{}
	h := handlers.NewChosen(
		&fakeRegistry{canHandle: true},
		&fakeChosenQueue{job: &jobs.Job{URL: "https://instagram.com/p/X/", Quality: "best"}},
		c,
		-1001234567890,
		52428800,
	)
	h.OverrideExtractor(&fakeFullExtractor{
		result: &extractors.VideoResult{DirectURL: "https://cdn/v.mp4", DurationSec: 5, SizeBytes: -1},
	})

	h.ProcessChosen(context.Background(), "job-id",
		func(_ *extractors.VideoResult, _ int64) (string, error) { return "cached-file-id", nil },
		func(_ string) error { return nil },
		func(_, _ string) error { return nil },
	)

	found := false
	for _, v := range c.stored {
		if v == "cached-file-id" {
			found = true
		}
	}
	if !found {
		t.Error("file_id was not cached after successful upload")
	}
}
