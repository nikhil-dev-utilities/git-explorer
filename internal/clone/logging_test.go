package clone

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

// captureLog swaps slog.Default() for one writing to a buffer, restoring the
// original on cleanup. slog.Default() is process-global state; this package's tests
// run sequentially (none call t.Parallel()), so swapping it for one test is safe.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return &buf
}

func TestRun_LogsStartAndSummary(t *testing.T) {
	buf := captureLog(t)
	target := t.TempDir()

	freshBare := newBareRepo(t)
	freshRepo := Repo{Org: "acme", Name: "fresh", CloneURL: freshBare}

	skippedRepo := Repo{Org: "acme", Name: "skipped", CloneURL: "https://example.invalid/acme/skipped.git"}
	initRepoWithOrigin(t, TargetPath(target, skippedRepo, false), "https://example.invalid/acme/skipped.git")

	Run(context.Background(), target, []Repo{freshRepo, skippedRepo}, false, 8)

	out := buf.String()
	if !strings.Contains(out, "clone run starting") {
		t.Errorf("log output = %q, want a \"clone run starting\" line", out)
	}
	if !strings.Contains(out, "clone run finished") {
		t.Errorf("log output = %q, want a \"clone run finished\" line", out)
	}
	if !strings.Contains(out, "cloned=1") {
		t.Errorf("log output = %q, want cloned=1", out)
	}
	if !strings.Contains(out, "skipped=1") {
		t.Errorf("log output = %q, want skipped=1", out)
	}
}

func TestRun_LogsEachFailureIndividually(t *testing.T) {
	buf := captureLog(t)
	target := t.TempDir()

	// A CloneURL git can't reach at all — the clone subprocess itself fails, not
	// just classification, giving a real per-Repo Err to log.
	failingRepo := Repo{Org: "acme", Name: "unreachable", CloneURL: filepath.Join(t.TempDir(), "does-not-exist.git")}

	Run(context.Background(), target, []Repo{failingRepo}, false, 8)

	out := buf.String()
	if !strings.Contains(out, "clone failed") {
		t.Errorf("log output = %q, want a \"clone failed\" line", out)
	}
	if !strings.Contains(out, "unreachable") {
		t.Errorf("log output = %q, want the failing repo's name", out)
	}
	if !strings.Contains(out, "failed=1") {
		t.Errorf("log output = %q, want failed=1 in the summary", out)
	}
}

func TestRun_NoFailureLinesWhenEverythingSucceeds(t *testing.T) {
	buf := captureLog(t)
	target := t.TempDir()
	freshBare := newBareRepo(t)

	Run(context.Background(), target, []Repo{{Org: "acme", Name: "fresh", CloneURL: freshBare}}, false, 8)

	out := buf.String()
	if strings.Contains(out, "clone failed") {
		t.Errorf("log output = %q, want no \"clone failed\" lines when nothing failed", out)
	}
	if !strings.Contains(out, "failed=0") {
		t.Errorf("log output = %q, want failed=0 in the summary", out)
	}
}
