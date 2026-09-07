// Package tui implements git-explorer's Bubble Tea shell: the two-pane Org/Repo
// browser, selection, clone dialog, and clone run. It depends on forge.Forge purely as
// an injected interface and never imports internal/forge/github — nor does it import
// internal/clone's execution path directly; clone execution and preview are injected
// as func values (see clone.go, added in a later slice) so this package's tests never
// touch a network or a subprocess. See DESIGN.md and ADR-0005/0006.
package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// Mode is the Model's explicit state, driving both Update dispatch and View
// rendering. Later slices of this PRD add CloneRun, HostSwitch, Fatal, and Help.
type Mode int

const (
	// ModeBrowse is the default mode: the Org and Repo panes, each with an
	// always-focused live filter.
	ModeBrowse Mode = iota
	// ModeLeavePrompt guards a non-empty Selection (ADR-0005): entered instead of
	// completing a navigation away from the Repo pane. clone now / discard / stay.
	ModeLeavePrompt
	// ModeCloneDialog is built out by a later slice of this PRD (#31); the mode
	// exists now so LeavePrompt's "clone now" choice has somewhere real to go.
	ModeCloneDialog
)

// Focus is which of the two Browse-mode panes is currently receiving key input.
type Focus int

const (
	FocusOrgs Focus = iota
	FocusRepos
)

// SortMode is the sort applied within a pane.
type SortMode int

const (
	SortByName SortMode = iota
	SortByActivity
)

// TriState is a hide/show/only cycle, used for the Repo pane's archived and fork
// facets.
type TriState int

const (
	TriHide TriState = iota
	TriShow
	TriOnly
)

// VisibilityFilter narrows the Repo pane by forge.Visibility, or shows every value.
type VisibilityFilter int

const (
	VisibilityFilterAll VisibilityFilter = iota
	VisibilityFilterPublic
	VisibilityFilterPrivate
	VisibilityFilterInternal
)

// AffiliationFilter narrows the Org pane by forge.Affiliation, or shows every value.
// This wasn't named as an acceptance criterion when this slice's issue was drafted,
// but the parent PRD's user story 8 explicitly asks for it alongside the Repo pane's
// facets ("the Org pane's Affiliation filter AND the Repo pane's archived/fork/
// visibility filters") — implemented here rather than left as a gap against the PRD.
type AffiliationFilter int

const (
	AffiliationFilterAll AffiliationFilter = iota
	AffiliationFilterOwner
	AffiliationFilterMember
	AffiliationFilterCollaborator
	AffiliationFilterNone
)

// Model is git-explorer's Bubble Tea model.
type Model struct {
	forge forge.Forge
	host  forge.Host

	mode  Mode
	focus Focus

	orgs       []forge.Org
	orgsCh     <-chan forge.OrgPage
	orgsLoaded bool
	// orgsErr and orgsFatalErr are recorded but not yet rendered distinctly — the
	// failure-surfaces slice of this PRD builds the Fatal / pane-scoped presentation
	// on top of these.
	orgsErr      error
	orgsFatalErr error

	orgFilter string
	orgCursor int
	orgSort   SortMode // no visible effect yet — Org has no activity field to
	// sort by; the key still updates this so pressing it in
	// either pane behaves consistently at the state level.
	orgAffiliation AffiliationFilter

	// currentOrg is the Org the Repo pane is (or was last) showing.
	currentOrg  forge.Org
	repos       []forge.Repo
	reposLoaded bool
	reposErr    error

	repoFilter     string
	repoCursor     int
	repoSort       SortMode
	archivedFilter TriState // default TriHide
	forkFilter     TriState // default TriHide
	visibility     VisibilityFilter

	// selected is the Selection: ticked Repo names within currentOrg. Per ADR-0005
	// a Selection belongs to exactly one Org and never spans Orgs — it is reset
	// whenever descend() starts a new Repo-pane session.
	selected map[string]bool
}

func (m Model) selectionCount() int {
	n := 0
	for _, v := range m.selected {
		if v {
			n++
		}
	}
	return n
}

// New constructs a Model. f is injected so this package's tests never depend on a
// real Forge implementation.
func New(f forge.Forge, host forge.Host) Model {
	return Model{
		forge: f,
		host:  host,
		mode:  ModeBrowse,
	}
}

func (m Model) Init() tea.Cmd {
	return listOrgsCmd(m.forge, m.host)
}

// backgroundCtx is used for commands dispatched from Update. A later slice of this
// PRD (cancellation for the Clone Run) introduces a cancellable context for that
// specific operation; Org/Repo loading has no cancellation requirement of its own.
func backgroundCtx() context.Context {
	return context.Background()
}
