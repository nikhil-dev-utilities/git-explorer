package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLumberjackWriter_ConfiguredFromLogConfig(t *testing.T) {
	w := newLumberjackWriter(LogConfig{Path: "/tmp/git-explorer.log", MaxSizeMB: 20})
	if w.Filename != "/tmp/git-explorer.log" {
		t.Errorf("Filename = %q, want /tmp/git-explorer.log", w.Filename)
	}
	if w.MaxSize != 20 {
		t.Errorf("MaxSize = %d, want 20 (from cfg.MaxSizeMB)", w.MaxSize)
	}
	if w.MaxBackups != 1 {
		t.Errorf("MaxBackups = %d, want 1", w.MaxBackups)
	}
	if w.Compress {
		t.Error("Compress = true, want false")
	}
}

func TestNewLogger_WritesToTheConfiguredFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "git-explorer.log")

	logger := NewLogger(LogConfig{Path: path, Level: "info", MaxSizeMB: 5})
	logger.Info("hello from a test")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if !strings.Contains(string(data), "hello from a test") {
		t.Errorf("log file content = %q, want it to contain the logged message", data)
	}
}

func TestNewLogger_UsesTextHandlerNotJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "git-explorer.log")

	logger := NewLogger(LogConfig{Path: path, Level: "info", MaxSizeMB: 5})
	logger.Info("a message", "key", "value")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	line := strings.TrimSpace(string(data))
	if strings.HasPrefix(line, "{") {
		t.Errorf("log line looks like JSON, want a text handler line: %q", line)
	}
	if !strings.Contains(line, "key=value") {
		t.Errorf("log line = %q, want slog's text-handler key=value shape", line)
	}
}

func TestNewLogger_OffLevelIsAWorkingNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "git-explorer.log")

	logger := NewLogger(LogConfig{Path: path, Level: "off", MaxSizeMB: 5})

	// Must not panic or error, at any level — off means off, not "off except when
	// something calls Error".
	logger.Debug("d")
	logger.Info("i")
	logger.Warn("w")
	logger.Error("e")

	if _, err := os.Stat(path); err == nil {
		t.Errorf("log file %s was created, want level:off to write nothing at all", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

// TestNewLogger_NeverWritesToStdoutOrStderr is the regression guard for this
// package's central invariant: this is a Bubble Tea program, and stdout/stderr are
// its render surface. It proves the invariant behaviorally — by actually capturing
// both streams while logging at every level — rather than only inspecting the
// constructed writer's type, since a future refactor could add a code path that
// silently starts writing to one of them without changing the writer's declared type.
func TestNewLogger_NeverWritesToStdoutOrStderr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "git-explorer.log")

	for _, level := range []string{"debug", "info", "warn", "error", "off"} {
		t.Run(level, func(t *testing.T) {
			stdoutR, stdoutW, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			stderrR, stderrW, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			origStdout, origStderr := os.Stdout, os.Stderr
			os.Stdout, os.Stderr = stdoutW, stderrW
			defer func() { os.Stdout, os.Stderr = origStdout, origStderr }()

			logger := NewLogger(LogConfig{Path: path, Level: level, MaxSizeMB: 5})
			logger.Debug("d")
			logger.Info("i")
			logger.Warn("w")
			logger.Error("e")

			stdoutW.Close()
			stderrW.Close()

			stdoutData, _ := io.ReadAll(stdoutR)
			stderrData, _ := io.ReadAll(stderrR)

			if len(stdoutData) != 0 {
				t.Errorf("level=%s wrote %d bytes to stdout, want 0: %q", level, len(stdoutData), stdoutData)
			}
			if len(stderrData) != 0 {
				t.Errorf("level=%s wrote %d bytes to stderr, want 0: %q", level, len(stderrData), stderrData)
			}
		})
	}
}
