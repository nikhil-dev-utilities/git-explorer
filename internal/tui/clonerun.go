package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

// CloneRunnerFunc executes a Clone Run. clone.Run has exactly this signature, so
// production wiring can pass it directly with no adapter; tests inject a fake.
//
// clone.Run is synchronous and non-streaming — it returns every Result together once
// the whole batch finishes or is cancelled, with no callback for per-Repo completions
// as they happen. A tea.Cmd wrapping this therefore delivers exactly one message, at
// the end. This package accepts that rather than reopening the already-merged
// internal/clone package for a progress callback, or duplicating its
// bounded-parallelism logic here to get real per-Repo messages (see #32's resolved
// discussion): while a run is in flight the view shows an indeterminate state, not
// per-Repo ticks, and switches straight to the full breakdown when the one message
// arrives.
type CloneRunnerFunc func(ctx context.Context, target string, repos []clone.Repo, orgSubdir bool, parallelism int) []clone.Result

// cloneRunResultMsg carries every Result together, in one message — see
// CloneRunnerFunc's doc comment for why.
type cloneRunResultMsg struct {
	results []clone.Result
}

// confirmCloneDialog starts a Clone Run for every Repo the dialog is currently
// previewing. Per ADR-0005, exactly one run is ever in flight — the Model has no
// field capable of representing two, and this is the only function that ever sets
// cloneRunInFlight true.
func (m Model) confirmCloneDialog() (Model, tea.Cmd) {
	repos := make([]clone.Repo, len(m.clonePreviewResults))
	for i, r := range m.clonePreviewResults {
		repos[i] = r.Repo
	}
	return m.startCloneRun(repos)
}

func (m Model) startCloneRun(repos []clone.Repo) (Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.mode = ModeCloneRun
	m.cloneRunCancel = cancel
	m.cloneRunResults = nil
	m.cloneRunInFlight = true

	fn := m.cloneRun
	target := m.cloneTarget
	orgSubdir := m.cloneOrgSubdir
	parallelism := m.cloneParallelism

	return m, func() tea.Msg {
		return cloneRunResultMsg{results: fn(ctx, target, repos, orgSubdir, parallelism)}
	}
}

func (m Model) handleCloneRunResult(msg cloneRunResultMsg) (Model, tea.Cmd) {
	m.cloneRunResults = msg.results
	m.cloneRunInFlight = false
	m.cloneRunCancel = nil
	return m, nil
}

// cancelCloneRun cancels the in-flight run's context. The Model stays in
// ModeCloneRun, still showing the in-flight state, until cloneRunResultMsg actually
// arrives — cancellation is cooperative, not instantaneous, but clone.Run (PRD 3) is
// already proven to unwind promptly once its context is done.
func (m Model) cancelCloneRun() Model {
	if m.cloneRunCancel != nil {
		m.cloneRunCancel()
	}
	return m
}

// retryFailures re-enters CloneRun with only the Repos whose prior attempt failed —
// not the full original Selection.
func (m Model) retryFailures() (Model, tea.Cmd) {
	var failed []clone.Repo
	for _, r := range m.cloneRunResults {
		if r.Err != nil {
			failed = append(failed, r.Repo)
		}
	}
	if len(failed) == 0 {
		return m, nil
	}
	return m.startCloneRun(failed)
}

// leaveCloneRunSummary returns to Browse once the run has finished. The Selection is
// cleared — the run has already acted on it, and leaving a stale "N selected"
// indicator around afterward would be confusing.
func (m Model) leaveCloneRunSummary() Model {
	m.mode = ModeBrowse
	m.selected = nil
	return m
}
