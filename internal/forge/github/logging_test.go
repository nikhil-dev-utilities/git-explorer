package github

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// captureLog swaps slog.Default() for one writing to a buffer at debug level (so
// even the success-path Debug log is captured), restoring the original on cleanup.
// slog.Default() is process-global state; this package's tests run sequentially
// (none call t.Parallel()), so swapping it for the duration of one test is safe.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return &buf
}

func TestRunAPI_LogsASuccessfulCall(t *testing.T) {
	buf := captureLog(t)
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "orgs/acme/repos", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte("[]"), ExitCode: 0},
	)
	a := newWithRunner(fr)

	if _, err := a.ListRepos(context.Background(), forge.Org{Name: "acme", Host: forge.Host{Name: "github.com"}}); err != nil {
		t.Fatalf("ListRepos() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "gh api call succeeded") {
		t.Errorf("log output = %q, want it to mention a successful gh api call", out)
	}
	if !strings.Contains(out, "listed repos") {
		t.Errorf("log output = %q, want the higher-level \"listed repos\" line too", out)
	}
}

func TestRunAPI_LogsAFailedCallWithoutLeakingStderrVerbatim(t *testing.T) {
	buf := captureLog(t)
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "orgs/acme/repos", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stderr: []byte("some secret-shaped error detail"), ExitCode: 1},
	)
	a := newWithRunner(fr)

	_, err := a.ListRepos(context.Background(), forge.Org{Name: "acme", Host: forge.Host{Name: "github.com"}})
	if err == nil {
		t.Fatal("ListRepos() error = nil, want the classified failure")
	}

	out := buf.String()
	if !strings.Contains(out, "gh api call failed") {
		t.Errorf("log output = %q, want it to mention the failed call", out)
	}
	// forge.ErrorKind has no String() method, so slog's text handler renders it as
	// its bare int value — ErrKindPaneScoped is 2, the classification a non-rate-
	// limit failure always gets (see classifyAPIFailure).
	if !strings.Contains(out, "kind=2") {
		t.Errorf("log output = %q, want kind=2 (ErrKindPaneScoped)", out)
	}
}

func TestCheckAuth_LogsOutcomeWithoutEverLoggingTheCredential(t *testing.T) {
	buf := captureLog(t)
	fr := newFakeRunner()
	fr.on(
		[]string{"auth", "token", "--hostname", "github.com"},
		runResult{Stdout: []byte("gho_supersecrettoken"), ExitCode: 0},
	)
	a := newWithRunner(fr)

	if err := a.checkAuth(context.Background(), forge.Host{Name: "github.com"}); err != nil {
		t.Fatalf("checkAuth() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "authenticated") {
		t.Errorf("log output = %q, want it to mention the authenticated outcome", out)
	}
	if strings.Contains(out, "supersecrettoken") {
		t.Fatalf("log output = %q, LEAKED THE CREDENTIAL — must never happen", out)
	}
}

func TestCheckAuth_LogsNotAuthenticated(t *testing.T) {
	buf := captureLog(t)
	fr := newFakeRunner()
	fr.on(
		[]string{"auth", "token", "--hostname", "github.com"},
		runResult{Stderr: []byte("not logged in"), ExitCode: 1},
	)
	a := newWithRunner(fr)

	if err := a.checkAuth(context.Background(), forge.Host{Name: "github.com"}); err == nil {
		t.Fatal("checkAuth() error = nil, want the not-authenticated failure")
	}

	out := buf.String()
	if !strings.Contains(out, "not authenticated") {
		t.Errorf("log output = %q, want it to mention not authenticated", out)
	}
}
