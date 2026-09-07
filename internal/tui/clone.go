package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

// ClonePreviewFunc classifies repos against target without cloning anything — a
// batched wrapper around clone.Classify. Its real implementation (a thin loop
// calling clone.Classify per Repo) is composition-root wiring, not part of this
// package: internal/tui only needs the type and, in tests, a fake satisfying it. No
// change to internal/clone itself was needed for this — it's a caller-side loop over
// the existing clone.Classify.
//
// clone.Result.Err here can only ever mean "classification itself failed" (e.g. git
// not installed) — never "the clone failed," since no clone is attempted by this
// func.
type ClonePreviewFunc func(ctx context.Context, target string, repos []clone.Repo, orgSubdir bool) []clone.Result

// clonePreviewMsg carries the result of a ClonePreviewFunc call.
type clonePreviewMsg struct {
	results []clone.Result
}

// enterCloneDialog transitions to ModeCloneDialog and dispatches the pre-flight
// classification for the current Selection. Both paths that can reach this mode —
// Enter on a non-empty Selection, and LeavePrompt's "clone now" — go through this one
// function, so neither can forget to actually compute the preview.
func (m Model) enterCloneDialog() (Model, tea.Cmd) {
	m.mode = ModeCloneDialog
	m.clonePreviewResults = nil
	return m, m.dispatchClonePreview()
}

func (m Model) dispatchClonePreview() tea.Cmd {
	fn := m.clonePreview
	target := m.cloneTarget
	repos := m.selectedCloneRepos()
	orgSubdir := m.cloneOrgSubdir
	return func() tea.Msg {
		return clonePreviewMsg{results: fn(backgroundCtx(), target, repos, orgSubdir)}
	}
}

// selectedCloneRepos converts the Selection (ticked names within m.repos) into
// clone.Repo values, resolving each CloneURL once via the injected Forge —
// internal/clone never computes one itself (ADR-0001 / PRD 3's own design).
func (m Model) selectedCloneRepos() []clone.Repo {
	var out []clone.Repo
	for _, r := range m.repos {
		if !m.selected[r.Name] {
			continue
		}
		out = append(out, clone.Repo{
			Org:      r.Org,
			Name:     r.Name,
			CloneURL: m.forge.CloneURL(r),
		})
	}
	return out
}

func (m Model) handleClonePreview(msg clonePreviewMsg) (Model, tea.Cmd) {
	m.clonePreviewResults = msg.results
	return m, nil
}

// toggleOrgSubdir flips the org-subdirectory toggle and recomputes the preview —
// every listed path must update live, per the PRD. It always starts off (TriHide-like
// default) and is never remembered between Clone Runs, per ADR-0007: New always
// constructs a Model with it false.
func (m Model) toggleOrgSubdir() (Model, tea.Cmd) {
	m.cloneOrgSubdir = !m.cloneOrgSubdir
	return m, m.dispatchClonePreview()
}

// leaveCloneDialog returns to Browse with focus on Repos and the Selection intact —
// Esc from the dialog clones nothing.
func (m Model) leaveCloneDialog() Model {
	m.mode = ModeBrowse
	return m
}
