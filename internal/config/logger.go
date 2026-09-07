package config

import (
	"io"
	"log/slog"
	"strings"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger constructs the application logger from resolved log settings (LogConfig,
// as returned by Load).
//
// This is a Bubble Tea program: stdout and stderr are the render surface, and a stray
// log write to either would corrupt the display. No code path here can return a
// logger backed by os.Stdout or os.Stderr — that isn't a preference, it's the reason a
// log file exists in this design at all. See DESIGN.md's "Logging" section.
//
// Output is bounded by size via lumberjack rather than growing without limit: one
// rotation (MaxBackups: 1) keeps recent content and drops the oldest, avoiding a
// read-rewrite-realign pass over a file that's open for append. The handler is text,
// not JSON — the audience is a human pasting output into an issue.
func NewLogger(cfg LogConfig) *slog.Logger {
	if strings.EqualFold(cfg.Level, "off") {
		// io.Discard, never os.Stdout/os.Stderr: a working no-op logger, not a nil
		// one callers have to guard against.
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	handler := slog.NewTextHandler(newLumberjackWriter(cfg), &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
	})
	return slog.New(handler)
}

// newLumberjackWriter is split out from NewLogger so the configuration decision it
// encodes (MaxBackups: 1, no compression, size from cfg) is directly inspectable by a
// test without needing to write megabytes of log lines to observe real rotation —
// rotation itself is lumberjack's own tested responsibility, not ours to re-test.
func newLumberjackWriter(cfg LogConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   cfg.Path,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: 1,
		Compress:   false,
	}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
