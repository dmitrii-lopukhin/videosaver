package insta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
)

type resolveRequest struct {
	URL     string `json:"url"`
	Audio   bool   `json:"audio"`
	Quality string `json:"quality"`
}

type resolveResponse struct {
	DirectURL    string `json:"direct_url"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
	DurationSec  int    `json:"duration_sec"`
	IsAudio      bool   `json:"is_audio"`
}

// Extractor calls the Python insta-resolver service to resolve Instagram URLs.
type Extractor struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string, timeout time.Duration) *Extractor {
	return &Extractor{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (e *Extractor) CanHandle(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.User != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "instagram.com" || strings.HasSuffix(host, ".instagram.com")
}

func (e *Extractor) Resolve(ctx context.Context, url string, opts extractors.ResolveOpts) (*extractors.VideoResult, error) {
	quality := opts.Quality
	if quality == "" {
		quality = "best"
	}

	body, _ := json.Marshal(resolveRequest{URL: url, Audio: opts.Audio, Quality: quality})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/resolve", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, extractors.ErrTimeout
		}
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var r resolveResponse
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return nil, fmt.Errorf("insta: decode response: %w", err)
		}
		return &extractors.VideoResult{
			DirectURL:    r.DirectURL,
			Title:        r.Title,
			ThumbnailURL: r.ThumbnailURL,
			DurationSec:  r.DurationSec,
			SizeBytes:    -1,
			IsAudio:      r.IsAudio,
		}, nil

	case http.StatusTooManyRequests:
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, &extractors.ErrRateLimited{RetryAfter: retryAfter}

	case http.StatusUnauthorized, http.StatusServiceUnavailable:
		return nil, extractors.ErrBlocked

	case http.StatusNotFound, http.StatusBadRequest:
		return nil, extractors.ErrNotFound

	default:
		return nil, fmt.Errorf("insta: unexpected status %d", resp.StatusCode)
	}
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 60 * time.Second
	}
	if secs, err := strconv.Atoi(header); err == nil {
		return time.Duration(secs) * time.Second
	}
	return 60 * time.Second
}
