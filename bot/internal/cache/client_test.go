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

func TestClient_SetJSONGetJSON(t *testing.T) {
	srv, _ := miniredis.Run()
	defer srv.Close()
	c, _ := New("redis://" + srv.Addr() + "/0")
	defer c.Close()
	ctx := context.Background()

	type payload struct {
		Name string
		N    int
	}
	in := payload{Name: "hello", N: 42}

	if err := c.SetJSON(ctx, "k", in, time.Minute); err != nil {
		t.Fatalf("SetJSON: %v", err)
	}

	var out payload
	found, err := c.GetJSON(ctx, "k", &out)
	if err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if !found {
		t.Fatal("GetJSON: key not found")
	}
	if out != in {
		t.Errorf("GetJSON = %+v, want %+v", out, in)
	}
}

func TestClient_GetJSON_Missing(t *testing.T) {
	srv, _ := miniredis.Run()
	defer srv.Close()
	c, _ := New("redis://" + srv.Addr() + "/0")
	defer c.Close()

	var v struct{ X int }
	found, err := c.GetJSON(context.Background(), "missing", &v)
	if err != nil {
		t.Fatalf("GetJSON(missing): %v", err)
	}
	if found {
		t.Error("GetJSON(missing) returned found=true")
	}
}

func TestClient_Lock(t *testing.T) {
	srv, _ := miniredis.Run()
	defer srv.Close()
	c, _ := New("redis://" + srv.Addr() + "/0")
	defer c.Close()
	ctx := context.Background()

	got, err := c.Lock(ctx, "mylock", 30*time.Second)
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if !got {
		t.Fatal("first Lock should return true")
	}

	got2, err := c.Lock(ctx, "mylock", 30*time.Second)
	if err != nil {
		t.Fatalf("Lock second: %v", err)
	}
	if got2 {
		t.Fatal("second Lock on same key should return false")
	}
}

func TestClient_Unlock(t *testing.T) {
	srv, _ := miniredis.Run()
	defer srv.Close()
	c, _ := New("redis://" + srv.Addr() + "/0")
	defer c.Close()
	ctx := context.Background()

	c.Lock(ctx, "lk", 30*time.Second)
	if err := c.Unlock(ctx, "lk"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	got, err := c.Lock(ctx, "lk", 30*time.Second)
	if err != nil {
		t.Fatalf("Lock after Unlock: %v", err)
	}
	if !got {
		t.Fatal("Lock after Unlock should return true")
	}
}

func TestVideoKey(t *testing.T) {
	k := VideoKey("https://instagram.com/p/ABC", false, "best")
	want := "video:https://instagram.com/p/ABC:audio=false:q=best"
	if k != want {
		t.Errorf("VideoKey = %q, want %q", k, want)
	}
}

func TestVideoKeyDefaultQuality(t *testing.T) {
	k := VideoKey("https://instagram.com/p/ABC", false, "")
	if k != "video:https://instagram.com/p/ABC:audio=false:q=best" {
		t.Errorf("unexpected key: %q", k)
	}
}

func TestVideoLockKey(t *testing.T) {
	k := VideoLockKey("https://instagram.com/p/ABC")
	want := "video:https://instagram.com/p/ABC:lock"
	if k != want {
		t.Errorf("VideoLockKey = %q, want %q", k, want)
	}
}
