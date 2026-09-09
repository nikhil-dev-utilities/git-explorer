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

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
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
	// ModeCloneDialog previews the destination and pre-flight Outcome for every
	// Repo in the current Selection.
	ModeCloneDialog
	// ModeCloneRun is modal (ADR-0005): exactly one run in flight at a time, pane
	// navigation blocked until it finishes or is cancelled.
	ModeCloneRun
	// ModeHostSwitch lists every configured Host, letting the user pick a new
	// active one without restarting the app.
	ModeHostSwitch
	// ModeFatal takes over the whole screen: nothing else in the app works until
	// it's fixed (not authenticated, gh/git missing), so nothing else is shown.
	ModeFatal
	// ModeHelp lists every binding from keymapTable — reachable from Browse only.
	ModeHelp
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
	// SortByAffiliation is Org-only — see orgSort's own doc comment.
	SortByAffiliation
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
	// hosts is every Host declared in the already-resolved Config this Model was
	// constructed with (internal/tui never loads config itself — see PRD 2). Must
	// be non-empty; New panics otherwise, since there is always at least the
	// implicit github.com Host per DESIGN.md's Config section.
	hosts         []forge.Host
	activeHostIdx int
	hostCursor    int // cursor within ModeHostSwitch's listing
	// hostsUserConfigured is false when hosts came from composition-root discovery
	// (gh's own authenticated-host list, or the last-resort implicit github.com)
	// rather than an explicit hosts: list the user wrote themselves. When false, a
	// Fatal "not authenticated" failure downgrades to pane-scoped instead of taking
	// over the whole screen — there's nothing the user configured wrong, just
	// nothing to discover yet, and pane-scoped's inline retry is the gentler fit.
	// An explicit hosts: list that fails auth is a real misconfiguration and stays
	// Fatal.
	hostsUserConfigured bool

	mode  Mode
	focus Focus

	// width and height come from the initial tea.WindowSizeMsg Bubble Tea always
	// sends at startup, and any subsequent terminal resize. Zero until the first
	// one arrives.
	width, height int

	orgs       []forge.Org
	orgsCh     <-chan forge.OrgPage
	orgsLoaded bool
	// orgsErr is a pane-scoped failure loading Orgs (some pages may have already
	// loaded and remain in orgs, untouched) — rendered inline in the Org pane.
	orgsErr error

	orgFilter string
	orgCursor int
	// orgSort cycles Name ⇄ Affiliation (issue #88) — Org has no activity-like field
	// the way Repo has PushedAt, so it gets its own two-state cycle
	// (nextOrgSortMode) rather than reusing Repo's Name ⇄ Activity one.
	orgSort        SortMode
	orgAffiliation AffiliationFilter

	// currentOrg is the Org the Repo pane is (or was last) showing.
	currentOrg  forge.Org
	repos       []forge.Repo
	reposLoaded bool
	// reposErr is a pane-scoped failure loading Repos for currentOrg, rendered
	// inline in the Repo pane.
	reposErr error

	// fatalErr takes over the whole screen (ModeFatal) — set from either Org or
	// Repo loading, whichever failed fatally.
	fatalErr error
	// transientErr renders in the status line only, from either pane, never
	// disturbing what's already shown.
	transientErr error

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

	// clonePreview classifies a Selection against cloneTarget without cloning
	// anything — injected so this package's tests never touch the filesystem or
	// git. clonePreviewResults holds the live result, recomputed whenever
	// cloneOrgSubdir changes.
	clonePreview        ClonePreviewFunc
	cloneTarget         string
	cloneOrgSubdir      bool // always starts false — never remembered, per ADR-0007
	clonePreviewResults []clone.Result

	// cloneRun executes a confirmed Clone Run. Per ADR-0005 exactly one is ever in
	// flight; cloneRunCancel is non-nil only while cloneRunInFlight is true.
	cloneRun         CloneRunnerFunc
	cloneParallelism int
	cloneRunInFlight bool
	cloneRunCancel   context.CancelFunc
	cloneRunResults  []clone.Result
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

// New constructs a Model. f, preview, and runner are injected so this package's
// tests never depend on a real Forge or touch the filesystem/git. hosts must be
// non-empty; the first is active at launch. hostsUserConfigured is whether hosts
// came from an explicit hosts: list the user wrote themselves, as opposed to
// composition-root discovery or its last-resort implicit-github.com fallback — see
// the Model field's own doc comment for what this changes. target pre-fills the
// clone dialog (from Config's clone.default_target — empty is valid and means the
// dialog opens with no default, exactly as DESIGN.md's zero-config case describes);
// it is never written back to anything, only ever read. parallelism bounds a Clone
// Run (Config's clone.parallelism); values below 1 are clone.Run's own concern, not
// this package's — it passes parallelism through unmodified.
func New(f forge.Forge, hosts []forge.Host, hostsUserConfigured bool, preview ClonePreviewFunc, runner CloneRunnerFunc, target string, parallelism int) Model {
	if len(hosts) == 0 {
		panic("tui.New: hosts must be non-empty")
	}
	return Model{
		forge:               f,
		hosts:               hosts,
		hostsUserConfigured: hostsUserConfigured,
		mode:                ModeBrowse,
		clonePreview:        preview,
		cloneTarget:         target,
		cloneRun:            runner,
		cloneParallelism:    parallelism,
	}
}

// activeHost is the Host currently being browsed.
func (m Model) activeHost() forge.Host {
	return m.hosts[m.activeHostIdx]
}

func (m Model) Init() tea.Cmd {
	return listOrgsCmd(m.forge, m.activeHost())
}

// backgroundCtx is used for commands dispatched from Update. A later slice of this
// PRD (cancellation for the Clone Run) introduces a cancellable context for that
// specific operation; Org/Repo loading has no cancellation requirement of its own.
func backgroundCtx() context.Context {
	return context.Background()
}
