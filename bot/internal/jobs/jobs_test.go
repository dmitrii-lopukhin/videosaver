package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/jobs"
)

func newQueue(t *testing.T) (*jobs.Queue, *miniredis.Miniredis) {
	t.Helper()
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	return jobs.NewQueue(rdb, 10*time.Minute), srv
}

func TestEnqueueDequeue(t *testing.T) {
	q, srv := newQueue(t)
	defer srv.Close()

	job := jobs.Job{
		URL:     "https://instagram.com/p/ABC/",
		UserID:  42,
		Audio:   false,
		Quality: "best",
	}

	id, err := q.Enqueue(context.Background(), job)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if id == "" {
		t.Fatal("Enqueue returned empty id")
	}

	got, err := q.Dequeue(context.Background(), id)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if got.URL != job.URL {
		t.Errorf("URL = %q, want %q", got.URL, job.URL)
	}
	if got.UserID != job.UserID {
		t.Errorf("UserID = %d, want %d", got.UserID, job.UserID)
	}
}

func TestDequeue_Missing(t *testing.T) {
	q, srv := newQueue(t)
	defer srv.Close()

	_, err := q.Dequeue(context.Background(), "nonexistent-id")
	if err != jobs.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	q, srv := newQueue(t)
	defer srv.Close()

	id, _ := q.Enqueue(context.Background(), jobs.Job{URL: "https://instagram.com/p/X/"})
	if err := q.Delete(context.Background(), id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := q.Dequeue(context.Background(), id)
	if err != jobs.ErrNotFound {
		t.Errorf("after Delete: expected ErrNotFound, got %v", err)
	}
}

func TestEnqueue_TTL(t *testing.T) {
	q, srv := newQueue(t)
	defer srv.Close()

	id, _ := q.Enqueue(context.Background(), jobs.Job{URL: "https://instagram.com/p/Y/"})
	srv.FastForward(11 * time.Minute)

	_, err := q.Dequeue(context.Background(), id)
	if err != jobs.ErrNotFound {
		t.Errorf("after TTL expiry: expected ErrNotFound, got %v", err)
	}
}
