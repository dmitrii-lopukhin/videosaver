package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a job ID does not exist (expired or never created).
var ErrNotFound = errors.New("jobs: not found")

// Job represents a deferred resolve+download task.
type Job struct {
	URL     string `json:"url"`
	UserID  int64  `json:"user_id"`
	Audio   bool   `json:"audio"`
	Quality string `json:"quality"`
}

// Queue stores and retrieves jobs in Redis.
type Queue struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewQueue(rdb *redis.Client, ttl time.Duration) *Queue {
	return &Queue{rdb: rdb, ttl: ttl}
}

// Enqueue stores the job and returns a unique ID.
func (q *Queue) Enqueue(ctx context.Context, job Job) (string, error) {
	id := uuid.NewString()
	b, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	if err := q.rdb.Set(ctx, jobKey(id), b, q.ttl).Err(); err != nil {
		return "", err
	}
	return id, nil
}

// Dequeue retrieves a job by ID without deleting it (caller calls Delete when done).
func (q *Queue) Dequeue(ctx context.Context, id string) (*Job, error) {
	b, err := q.rdb.Get(ctx, jobKey(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var job Job
	if err := json.Unmarshal(b, &job); err != nil {
		return nil, fmt.Errorf("jobs: decode: %w", err)
	}
	return &job, nil
}

// Delete removes a job from Redis after it has been processed.
func (q *Queue) Delete(ctx context.Context, id string) error {
	return q.rdb.Del(ctx, jobKey(id)).Err()
}

func jobKey(id string) string {
	return "job:" + id
}
