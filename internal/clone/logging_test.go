package clone

import (
	"bytes"
	"context"
	"log/slog"
	"os"
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

// TestRun_LogsProgressPerRepo is the regression test for the feature this file
// exists to prove: a "clone failed" summary only appeared after the whole batch
// finished, and nothing at all was logged for Skipped/Conflict/successfully-cloned
// Repos — tailing the log during a long run showed silence, then one final line.
// Every Repo must now get its own line the moment its outcome is known.
func TestRun_LogsProgressPerRepo(t *testing.T) {
	buf := captureLog(t)
	target := t.TempDir()

	freshBare := newBareRepo(t)
	freshRepo := Repo{Org: "acme", Name: "fresh", CloneURL: freshBare}

	skippedRepo := Repo{Org: "acme", Name: "skipped", CloneURL: "https://example.invalid/acme/skipped.git"}
	initRepoWithOrigin(t, TargetPath(target, skippedRepo, false), "https://example.invalid/acme/skipped.git")

	conflictRepo := Repo{Org: "acme", Name: "conflict", CloneURL: "https://example.invalid/acme/conflict.git"}
	if err := os.MkdirAll(TargetPath(target, conflictRepo, false), 0o755); err != nil {
		t.Fatalf("seeding a conflicting non-git directory: %v", err)
	}

	Run(context.Background(), target, []Repo{freshRepo, skippedRepo, conflictRepo}, false, 8)

	out := buf.String()
	if n := strings.Count(out, "clone progress"); n != 3 {
		t.Fatalf("log output = %q, want exactly 3 \"clone progress\" lines (one per Repo), got %d", out, n)
	}
	for _, name := range []string{"fresh", "skipped", "conflict"} {
		if !strings.Contains(out, "repo="+name) {
			t.Errorf("log output = %q, want a progress line for repo=%s", out, name)
		}
	}
}

// TestRun_FailureIsLoggedExactlyOnce guards against the summary loop this feature
// removed being reintroduced alongside the new live per-Repo logging — a failure
// must appear once, not twice.
func TestRun_FailureIsLoggedExactlyOnce(t *testing.T) {
	buf := captureLog(t)
	target := t.TempDir()
	failingRepo := Repo{Org: "acme", Name: "unreachable", CloneURL: filepath.Join(t.TempDir(), "does-not-exist.git")}

	Run(context.Background(), target, []Repo{failingRepo}, false, 8)

	out := buf.String()
	if n := strings.Count(out, "clone failed"); n != 1 {
		t.Fatalf("log output = %q, want exactly one \"clone failed\" line, got %d", out, n)
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
