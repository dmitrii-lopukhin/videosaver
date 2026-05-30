package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestClient_PingSetGet(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer srv.Close()

	c, err := New("redis://" + srv.Addr() + "/0")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	if err := c.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "v" {
		t.Errorf("Get returned %q, want %q", got, "v")
	}
}

func TestClient_GetMissingReturnsEmpty(t *testing.T) {
	srv, _ := miniredis.Run()
	defer srv.Close()

	c, _ := New("redis://" + srv.Addr() + "/0")
	defer c.Close()

	got, err := c.Get(context.Background(), "missing")
	if err != nil {
		t.Errorf("Get(missing) error: %v", err)
	}
	if got != "" {
		t.Errorf("Get(missing) = %q, want empty", got)
	}
}
