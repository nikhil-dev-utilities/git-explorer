package tui

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
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

func TestApplyOrgsFailure_LogsTheClassifiedKind(t *testing.T) {
	buf := captureLog(t)
	f := &fakeForge{orgsErr: &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated"}}
	tm := newTestModel(t, f) // hostsUserConfigured: true — stays Fatal
	_ = finalModelAfter(t, tm)

	out := buf.String()
	if !strings.Contains(out, "org load failure") {
		t.Errorf("log output = %q, want an \"org load failure\" line", out)
	}
	// ErrorKind has no String() method — logged as its bare int value.
	// ErrKindFatal is 0.
	if !strings.Contains(out, "kind=0") {
		t.Errorf("log output = %q, want kind=0 (ErrKindFatal)", out)
	}
}

func TestApplyReposFailure_LogsTheClassifiedKind(t *testing.T) {
	buf := captureLog(t)
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		reposErr: &forge.Error{Kind: forge.ErrKindPaneScoped, Message: "boom"},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend into acme -> ListRepos -> reposErr
	_ = finalModelAfter(t, tm)

	out := buf.String()
	if !strings.Contains(out, "repo load failure") {
		t.Errorf("log output = %q, want a \"repo load failure\" line", out)
	}
	// ErrKindPaneScoped is 2.
	if !strings.Contains(out, "kind=2") {
		t.Errorf("log output = %q, want kind=2 (ErrKindPaneScoped)", out)
	}
}

// A discovered (not user-configured) Host's Fatal auth failure downgrades to
// PaneScoped before it's logged — the log should reflect what actually happened
// (PaneScoped), not the pre-downgrade classification, since that's what the rest of
// the Model acted on too.
func TestApplyOrgsFailure_LogsTheDowngradedKindForDiscoveredHosts(t *testing.T) {
	buf := captureLog(t)
	f := &fakeForge{orgsErr: &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated"}}
	tm := newTestModelWithDiscoveredHosts(t, f)
	_ = finalModelAfter(t, tm)

	out := buf.String()
	if !strings.Contains(out, "kind=2") {
		t.Errorf("log output = %q, want kind=2 (ErrKindPaneScoped, post-downgrade)", out)
	}
	if strings.Contains(out, "kind=0") {
		t.Errorf("log output = %q, want no kind=0 — the pre-downgrade Fatal kind should never be logged as what happened", out)
	}
}
