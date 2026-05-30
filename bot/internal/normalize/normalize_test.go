package normalize_test

import (
	"testing"

	"github.com/dmitrii-lopukhin/videosaver/bot/internal/normalize"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			"https://www.instagram.com/reels/DY2WsLGIgML/",
			"https://www.instagram.com/reels/DY2WsLGIgML",
		},
		{
			"https://www.instagram.com/p/ABC123/?igshid=xyz&utm=foo",
			"https://www.instagram.com/p/ABC123",
		},
		{
			"https://INSTAGRAM.COM/reel/ABC/",
			"https://instagram.com/reel/ABC",
		},
		{
			"https://www.instagram.com/p/ABC123/#anchor",
			"https://www.instagram.com/p/ABC123",
		},
	}

	for _, tc := range cases {
		got, err := normalize.Normalize(tc.in)
		if err != nil {
			t.Errorf("Normalize(%q) error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeInvalidURL(t *testing.T) {
	_, err := normalize.Normalize("://not a url")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}
