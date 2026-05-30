package download_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/download"
)

func TestFetch_Success(t *testing.T) {
	payload := "hello video bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	rc, err := download.Fetch(context.Background(), srv.URL, int64(len(payload)+100))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	defer rc.Close()

	got, _ := io.ReadAll(rc)
	if string(got) != payload {
		t.Errorf("got %q, want %q", got, payload)
	}
}

func TestFetch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := download.Fetch(context.Background(), srv.URL, 1024)
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}

func TestFetch_ContentLengthOverLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strings.Repeat("x", 1000)))
	}))
	defer srv.Close()

	_, err := download.Fetch(context.Background(), srv.URL, 500)
	if err != download.ErrTooLarge {
		t.Errorf("expected ErrTooLarge, got %v", err)
	}
}

func TestFetch_StreamOverLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strings.Repeat("x", 200)))
	}))
	defer srv.Close()

	rc, err := download.Fetch(context.Background(), srv.URL, 100)
	if err != nil {
		if err == download.ErrTooLarge {
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	_, readErr := io.ReadAll(rc)
	if readErr != download.ErrTooLarge {
		t.Errorf("expected ErrTooLarge while reading, got %v", readErr)
	}
}
