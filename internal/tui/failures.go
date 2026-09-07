package tui

import (
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// classify reads the Kind a forge.Error already carries, rather than re-deriving a
// classification from string matching. A non-forge.Error (shouldn't happen in
// practice — every Forge implementation is documented to return one — but this
// package depends only on the interface) is treated as pane-scoped: the
// conservative choice, since it neither halts the whole app (Fatal) nor claims to
// resolve itself (Transient).
func classify(err error) forge.ErrorKind {
	var fe *forge.Error
	if errors.As(err, &fe) {
		return fe.Kind
	}
	return forge.ErrKindPaneScoped
}

func retryAfter(err error) time.Duration {
	var fe *forge.Error
	if errors.As(err, &fe) {
		return fe.RetryAfter
	}
	return 0
}

// applyOrgsFailure routes an error from either the synchronous ListOrgs call or a
// page's own Err by its Kind: Fatal takes over the whole screen; Transient renders in
// the status line only, without disturbing anything already loaded; PaneScoped
// renders inline in the Org pane, also preserving whatever already loaded. In every
// case, orgsLoaded is set so the pane stops showing "loading" — the stream has ended
// either way, per Forge's contract.
func (m Model) applyOrgsFailure(err error) Model {
	m.orgsLoaded = true
	switch classify(err) {
	case forge.ErrKindFatal:
		m.mode = ModeFatal
		m.fatalErr = err
	case forge.ErrKindTransient:
		m.transientErr = err
	default:
		m.orgsErr = err
	}
	return m
}

func (m Model) applyReposFailure(err error) Model {
	m.reposLoaded = true
	switch classify(err) {
	case forge.ErrKindFatal:
		m.mode = ModeFatal
		m.fatalErr = err
	case forge.ErrKindTransient:
		m.transientErr = err
	default:
		m.reposErr = err
	}
	return m
}

// retryOrgs re-attempts ListOrgs from scratch. Forge has no "resume from the page
// that failed" API, so a retry is a clean restart: whatever loaded before the
// failure is discarded here, at the moment the user asks to retry — not
// automatically when the failure first arrived, which is what actually protects it
// per DESIGN.md's "keep whatever already loaded" until the user acts.
func (m Model) retryOrgs() (Model, tea.Cmd) {
	m.orgs = nil
	m.orgsCh = nil
	m.orgsLoaded = false
	m.orgsErr = nil
	m.transientErr = nil
	return m, listOrgsCmd(m.forge, m.activeHost())
}

// retryRepos re-attempts ListRepos for currentOrg from scratch, the same way
// retryOrgs restarts Org loading.
func (m Model) retryRepos() (Model, tea.Cmd) {
	m.repos = nil
	m.reposLoaded = false
	m.reposErr = nil
	m.transientErr = nil
	return m, listReposCmd(m.forge, m.currentOrg)
}

// retry is ^r: retries whichever pane is focused and actually has a pane-scoped
// failure to retry. It is a no-op otherwise, rather than an error — pressing retry
// with nothing to retry is harmless.
func (m Model) retry() (Model, tea.Cmd) {
	switch m.focus {
	case FocusOrgs:
		if m.orgsErr != nil {
			return m.retryOrgs()
		}
	case FocusRepos:
		if m.reposErr != nil {
			return m.retryRepos()
		}
	}
	return m, nil
}
