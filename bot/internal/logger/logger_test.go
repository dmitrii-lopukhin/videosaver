package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestNew_HonorsLevel(t *testing.T) {
	var buf bytes.Buffer
	log := New("warn", &buf)

	log.Info().Msg("should not appear")
	log.Warn().Msg("should appear")

	out := buf.String()
	if strings.Contains(out, "should not appear") {
		t.Errorf("info log leaked at warn level: %q", out)
	}
	if !strings.Contains(out, "should appear") {
		t.Errorf("warn log missing: %q", out)
	}
}

func TestNew_InvalidLevelDefaultsToInfo(t *testing.T) {
	var buf bytes.Buffer
	log := New("garbage-level", &buf)

	log.Info().Msg("hello")
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("expected info-level output, got: %q", buf.String())
	}
	var _ zerolog.Logger = log
}
