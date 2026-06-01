package handlers

import (
	"context"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/extractors"
)

// ResolveCachedForTest exposes resolveCached to external tests.
func (h *InlineHandler) ResolveCachedForTest(ctx context.Context, norm string) (extractors.VideoResult, error) {
	return h.resolveCached(ctx, norm)
}
