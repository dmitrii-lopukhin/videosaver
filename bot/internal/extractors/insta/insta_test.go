package insta_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors/insta"
)

func serve(code int, body any, headers map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(body)
	}))
}

func TestCanHandle(t *testing.T) {
	e := insta.New("http://localhost", 5*time.Second)
	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.instagram.com/reels/ABC/", true},
		{"https://instagram.com/p/XYZ/", true},
		{"https://youtube.com/watch?v=abc", false},
		{"https://tiktok.com/@user/video/123", false},
	}
	for _, tc := range cases {
		if got := e.CanHandle(tc.url); got != tc.want {
			t.Errorf("CanHandle(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

func TestResolve_Success(t *testing.T) {
	srv := serve(200, map[string]any{
		"direct_url":    "https://cdn/v.mp4",
		"title":         "cool video",
		"thumbnail_url": "https://cdn/t.jpg",
		"duration_sec":  36,
		"is_audio":      false,
	}, nil)
	defer srv.Close()

	e := insta.New(srv.URL, 5*time.Second)
	r, err := e.Resolve(context.Background(), "https://instagram.com/reels/ABC/", extractors.ResolveOpts{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.DirectURL != "https://cdn/v.mp4" {
		t.Errorf("DirectURL = %q", r.DirectURL)
	}
	if r.Title != "cool video" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.DurationSec != 36 {
		t.Errorf("DurationSec = %d", r.DurationSec)
	}
}

func TestResolve_400_ErrNotFound(t *testing.T) {
	srv := serve(400, map[string]string{"detail": "unsupported url"}, nil)
	defer srv.Close()
	e := insta.New(srv.URL, 5*time.Second)
	_, err := e.Resolve(context.Background(), "https://instagram.com/p/X/", extractors.ResolveOpts{})
	if err != extractors.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestResolve_401_ErrBlocked(t *testing.T) {
	srv := serve(401, map[string]string{"detail": "session expired"}, nil)
	defer srv.Close()
	e := insta.New(srv.URL, 5*time.Second)
	_, err := e.Resolve(context.Background(), "https://instagram.com/p/X/", extractors.ResolveOpts{})
	if err != extractors.ErrBlocked {
		t.Errorf("expected ErrBlocked, got %v", err)
	}
}

func TestResolve_404_ErrNotFound(t *testing.T) {
	srv := serve(404, map[string]string{"detail": "not found"}, nil)
	defer srv.Close()
	e := insta.New(srv.URL, 5*time.Second)
	_, err := e.Resolve(context.Background(), "https://instagram.com/p/X/", extractors.ResolveOpts{})
	if err != extractors.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestResolve_429_ErrRateLimited(t *testing.T) {
	srv := serve(429, map[string]string{"detail": "rate limited"}, map[string]string{"Retry-After": "120"})
	defer srv.Close()
	e := insta.New(srv.URL, 5*time.Second)
	_, err := e.Resolve(context.Background(), "https://instagram.com/p/X/", extractors.ResolveOpts{})
	rl, ok := err.(*extractors.ErrRateLimited)
	if !ok {
		t.Fatalf("expected *ErrRateLimited, got %T: %v", err, err)
	}
	if rl.RetryAfter != 120*time.Second {
		t.Errorf("RetryAfter = %s, want 120s", rl.RetryAfter)
	}
}

func TestResolve_503_ErrBlocked(t *testing.T) {
	srv := serve(503, map[string]string{"detail": "no sessions loaded"}, nil)
	defer srv.Close()
	e := insta.New(srv.URL, 5*time.Second)
	_, err := e.Resolve(context.Background(), "https://instagram.com/p/X/", extractors.ResolveOpts{})
	if err != extractors.ErrBlocked {
		t.Errorf("expected ErrBlocked, got %v", err)
	}
}

func TestResolve_500_Error(t *testing.T) {
	srv := serve(500, map[string]string{"detail": "resolver error"}, nil)
	defer srv.Close()
	e := insta.New(srv.URL, 5*time.Second)
	_, err := e.Resolve(context.Background(), "https://instagram.com/p/X/", extractors.ResolveOpts{})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if err == extractors.ErrNotFound || err == extractors.ErrBlocked {
		t.Errorf("500 should not map to sentinel errors, got %v", err)
	}
}

// Compile-time check: insta.Extractor implements extractors.Extractor.
var _ extractors.Extractor = (*insta.Extractor)(nil)
