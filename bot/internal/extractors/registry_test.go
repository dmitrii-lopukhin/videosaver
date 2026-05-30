package extractors_test

import (
	"context"
	"testing"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
)

type containsExtractor struct{ needle string }

func (c *containsExtractor) CanHandle(url string) bool {
	for i := 0; i+len(c.needle) <= len(url); i++ {
		if url[i:i+len(c.needle)] == c.needle {
			return true
		}
	}
	return false
}
func (c *containsExtractor) Resolve(_ context.Context, _ string, _ extractors.ResolveOpts) (*extractors.VideoResult, error) {
	return &extractors.VideoResult{Title: c.needle}, nil
}

func TestRegistry_For_Match(t *testing.T) {
	ig := &containsExtractor{"instagram.com"}
	yt := &containsExtractor{"youtube.com"}
	reg := extractors.NewRegistry(ig, yt)

	e, ok := reg.For("https://www.instagram.com/p/ABC/")
	if !ok {
		t.Fatal("expected match for instagram URL")
	}
	if e != ig {
		t.Error("wrong extractor returned")
	}
}

func TestRegistry_For_NoMatch(t *testing.T) {
	ig := &containsExtractor{"instagram.com"}
	reg := extractors.NewRegistry(ig)

	_, ok := reg.For("https://tiktok.com/@u/video/1")
	if ok {
		t.Error("expected no match for tiktok URL")
	}
}

func TestRegistry_For_FirstMatchWins(t *testing.T) {
	a := &containsExtractor{"instagram.com"}
	b := &containsExtractor{"instagram.com"}
	reg := extractors.NewRegistry(a, b)

	e, ok := reg.For("https://instagram.com/p/X/")
	if !ok {
		t.Fatal("expected match")
	}
	if e != a {
		t.Error("expected first extractor to win")
	}
}
