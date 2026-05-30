//go:build ignore

// Smoke test: resolve an Instagram URL and download the video to disk.
// Usage: go run ./cmd/smoketest <instagram_url>
//
// Env vars: INSTA_RESOLVER_URL (default http://localhost:8000),
//
//	INSTA_RESOLVER_TIMEOUT_SEC (default 30),
//	DOWNLOAD_MAX_BYTES (default 52428800).
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/download"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors/insta"
	"github.com/dmitrii-lopukhin/videosaver/bot/internal/normalize"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/smoketest <instagram_url>")
		os.Exit(1)
	}

	rawURL := os.Args[1]
	resolverURL := getenv("INSTA_RESOLVER_URL", "http://localhost:8000")
	timeoutSec := getenvInt("INSTA_RESOLVER_TIMEOUT_SEC", 30)
	maxBytes := getenvInt64("DOWNLOAD_MAX_BYTES", 52428800)

	normURL, err := normalize.Normalize(rawURL)
	if err != nil {
		fatalf("normalize: %v", err)
	}
	fmt.Println("normalized:", normURL)

	e := insta.New(resolverURL, time.Duration(timeoutSec)*time.Second)
	if !e.CanHandle(normURL) {
		fatalf("unsupported URL (not instagram.com)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	fmt.Println("resolving via insta-resolver…")
	result, err := e.Resolve(ctx, normURL, extractors.ResolveOpts{Quality: "best"})
	if err != nil {
		fatalf("resolve: %v", err)
	}
	fmt.Printf("title:       %s\n", result.Title)
	fmt.Printf("duration:    %ds\n", result.DurationSec)
	fmt.Printf("direct_url:  %s\n", result.DirectURL)

	fmt.Println("downloading…")
	rc, err := download.Fetch(ctx, result.DirectURL, maxBytes)
	if err != nil {
		fatalf("download: %v", err)
	}
	defer rc.Close()

	outPath := "smoketest_out.mp4"
	f, err := os.Create(outPath)
	if err != nil {
		fatalf("create output file: %v", err)
	}
	defer f.Close()

	n, err := io.Copy(f, rc)
	if err != nil {
		fatalf("write: %v", err)
	}
	fmt.Printf("saved %d bytes to %s\n", n, outPath)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getenvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}
