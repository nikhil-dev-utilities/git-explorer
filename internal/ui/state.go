package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	glyph "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// footerRows are the rows under the panes: host line, status line, key hints.
const footerRows = 3

type mode int

const (
	modeBrowse mode = iota
	modeOptions
	modeLeave
	modeHost
	modeHelp
	modeFatal
	modeClone
)

type focus int

const (
	focusOrgs focus = iota
	focusRepos
)

// Deps is everything the UI needs from outside; it never imports a Forge implementation.
type Deps struct {
	Forge               forge.Forge
	Hosts               []forge.Host
	HostsUserConfigured bool
	CloneTarget         string
	Parallelism         int
}

var (
	focusedBorder = glyph.RGB(0x5f, 0x87, 0xff)
	blurredBorder = glyph.RGB(0x58, 0x58, 0x58)
)

// state is mutated only on Glyph's render goroutine: key handlers, and closures queued
// through apply, so Glyph (which reads these fields by pointer every frame) never races.
type state struct {
	d Deps

	mode  mode
	focus focus

	hosts      []forge.Host
	hostIdx    int
	hostCursor int

	orgAll     []forge.Org
	orgRows    []orgRow
	orgAff     affFilter
	orgSort    sortMode
	orgsLoaded bool
	orgsErr    error
	orgGen     int
	orgCancel  context.CancelFunc
	orgFL      *glyph.FilterListC[orgRow]
	orgStream  *glyph.StreamWriter[orgRow]

	currentOrg  forge.Org
	repoAll     []forge.Repo
	repoRows    []repoRow
	archived    tri
	fork        tri
	vis         visFilter
	repoSort    sortMode
	reposLoaded bool
	reposErr    error
	repoGen     int
	repoFL      *glyph.FilterListC[repoRow]
	selected    map[string]bool

	fatalErr     error
	transientErr error

	// filter text per pane, edited by the always-focused text handler
	query [2]string
	cur   [2]int
	text  [2]*riffkey.TextHandler

	width, height int
	orgWidthIdx   int
	lay           layout

	// derived display state, refreshed by sync
	orgBorder, repoBorder glyph.Color
	orgW, paneH           int16
	orgTitle, repoTitle   string
	orgChips, repoChips   string
	orgMsg, repoMsg       string
	footer, hint, status  string
	fatalText             string
	showBrowse            bool
	showTooNarrow         bool
	showFatal             bool
	showOptions           bool
	showLeave             bool
	showHost              bool
	showHelp              bool
	showClone             bool

	now func() time.Time

	// seams: production wires these to Glyph; tests make them synchronous.
	spawn   func(func())
	apply   func(func())
	refresh func()
	// onMode lets the view push/pop the router that owns keys for a modal mode.
	onMode func(prev, next mode)
}

func newState(d Deps, spawn, apply func(func()), refresh func()) *state {
	if len(d.Hosts) == 0 {
		panic("ui: Deps.Hosts must be non-empty")
	}
	s := &state{
		d:           d,
		hosts:       d.Hosts,
		orgWidthIdx: defaultOrgWidthIdx,
		archived:    triHide,
		fork:        triHide,
		selected:    map[string]bool{},
		width:       100,
		height:      30,
		now:         time.Now,
		spawn:       spawn,
		apply:       apply,
		refresh:     refresh,
	}
	s.orgFL = glyph.FilterList(&s.orgRows, func(r *orgRow) string { return r.Org.Name })
	s.repoFL = glyph.FilterList(&s.repoRows, func(r *repoRow) string { return r.Repo.Name })
	for i := range s.text {
		i := i
		s.text[i] = riffkey.NewTextHandler(&s.query[i], &s.cur[i])
		s.text[i].OnChange = func(q string) { s.setQuery(focus(i), q) }
	}
	s.resize(s.width, s.height)
	s.loadOrgs()
	return s
}

func (s *state) activeHost() forge.Host { return s.hosts[s.hostIdx] }

// pane is the slice of a FilterList both panes share.
type pane interface {
	SetQuery(string)
	Clear()
	SelectNext()
	SelectPrev()
	PageDown()
	PageUp()
}

func (s *state) pane(f focus) pane {
	if f == focusOrgs {
		return s.orgFL
	}
	return s.repoFL
}

func (s *state) setQuery(f focus, q string) {
	s.query[f], s.cur[f] = q, len(q)
	s.pane(f).SetQuery(q)
	s.sync()
}

func (s *state) clearQuery(f focus) {
	s.query[f], s.cur[f] = "", 0
	s.pane(f).Clear()
	s.sync()
}

// handleText edits the focused pane's filter (ADR-0006: the filter is always live).
func (s *state) handleText(k riffkey.Key) bool {
	if s.mode != modeBrowse {
		return false
	}
	return s.text[s.focus].HandleKey(k)
}

func (s *state) resize(w, h int) {
	s.width, s.height = w, h
	s.lay = layoutFor(w, s.orgWidthIdx)
	s.sync()
}

func (s *state) cycleOrgWidth() {
	s.orgWidthIdx = (s.orgWidthIdx + 1) % len(orgPaneWidths)
	s.lay = layoutFor(s.width, s.orgWidthIdx)
	s.sync()
}

func (s *state) setMode(next mode) {
	prev := s.mode
	s.mode = next
	s.sync()
	if s.onMode != nil && prev != next {
		s.onMode(prev, next)
	}
}

// ---- loading ---------------------------------------------------------------------

// loadOrgs restarts Org loading from scratch: Forge has no resume, and a load never
// merges with what an earlier host or attempt showed (ADR-0007).
func (s *state) loadOrgs() {
	if s.orgCancel != nil {
		s.orgCancel()
	}
	s.orgGen++
	gen := s.orgGen
	ctx, cancel := context.WithCancel(context.Background())
	s.orgCancel = cancel

	s.orgAll, s.orgsLoaded, s.orgsErr, s.fatalErr, s.transientErr = nil, false, nil, nil, nil
	s.rebuildOrgs()
	if s.orgStream != nil {
		s.orgStream.Close()
	}
	s.orgStream = s.orgFL.Stream(s.refresh)

	host := s.activeHost()
	s.spawn(func() {
		ch, err := s.d.Forge.ListOrgs(ctx, host)
		if err != nil {
			s.apply(func() {
				if gen == s.orgGen {
					s.orgsFailed(err)
				}
			})
			return
		}
		for page := range ch {
			page := page
			if page.Err != nil {
				s.apply(func() {
					if gen == s.orgGen {
						s.orgsFailed(page.Err)
					}
				})
				return
			}
			s.apply(func() {
				if gen != s.orgGen {
					return
				}
				s.orgAll = append(s.orgAll, page.Orgs...)
				s.rebuildOrgs()
			})
		}
		s.apply(func() {
			if gen != s.orgGen {
				return
			}
			s.orgsLoaded = true
			s.orgStream.Close()
			s.sync()
		})
	})
}

// loadRepos fetches the current Org's Repos, lazily and never cached.
func (s *state) loadRepos() {
	s.repoGen++
	gen := s.repoGen
	org := s.currentOrg
	s.repoAll, s.reposLoaded, s.reposErr, s.transientErr = nil, false, nil, nil
	s.rebuildRepos()
	s.spawn(func() {
		repos, err := s.d.Forge.ListRepos(context.Background(), org)
		s.apply(func() {
			if gen != s.repoGen {
				return
			}
			if err != nil {
				s.reposFailed(err)
				return
			}
			s.repoAll, s.reposLoaded = repos, true
			s.rebuildRepos()
		})
	})
}

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

// errorKind is classify, except a Fatal is downgraded to pane-scoped when the Host came
// from discovery rather than the user's own config: nothing was misconfigured, there was
// just nothing to find yet.
func (s *state) errorKind(err error) forge.ErrorKind {
	kind := classify(err)
	if kind == forge.ErrKindFatal && !s.d.HostsUserConfigured {
		return forge.ErrKindPaneScoped
	}
	return kind
}

func (s *state) orgsFailed(err error) {
	s.orgsLoaded = true
	if s.orgStream != nil {
		s.orgStream.Close()
	}
	kind := s.errorKind(err)
	slog.Warn("org load failure", "kind", kind, "error", err)
	switch kind {
	case forge.ErrKindFatal:
		s.fatalErr = err
		s.setMode(modeFatal)
	case forge.ErrKindTransient:
		s.transientErr = err
	default:
		s.orgsErr = err
	}
	s.sync()
}

func (s *state) reposFailed(err error) {
	s.reposLoaded = true
	kind := s.errorKind(err)
	slog.Warn("repo load failure", "kind", kind, "error", err)
	switch kind {
	case forge.ErrKindFatal:
		s.fatalErr = err
		s.setMode(modeFatal)
	case forge.ErrKindTransient:
		s.transientErr = err
	default:
		s.reposErr = err
	}
	s.sync()
}

// reload re-attempts the focused pane from scratch, keeping nothing.
func (s *state) reload() {
	if s.focus == focusRepos && s.currentOrg.Name != "" {
		s.loadRepos()
		return
	}
	s.loadOrgs()
}

// ---- rows ------------------------------------------------------------------------

func (s *state) rebuildOrgs() {
	orgs := sortedOrgs(s.orgAll, s.orgAff, s.orgSort)
	rows := make([]orgRow, len(orgs))
	for i, o := range orgs {
		rows[i] = newOrgRow(o)
	}
	s.orgRows = rows
	s.orgFL.Refresh()
	s.sync()
}

func (s *state) rebuildRepos() {
	repos := sortedRepos(s.repoAll, s.archived, s.fork, s.vis, s.repoSort)
	now := s.now()
	rows := make([]repoRow, len(repos))
	for i, r := range repos {
		rows[i] = newRepoRow(r, s.selected[r.Name], now)
	}
	s.repoRows = rows
	s.repoFL.Refresh()
	s.sync()
}

// ---- navigation and selection ----------------------------------------------------

func (s *state) selectionCount() int {
	n := 0
	for _, v := range s.selected {
		if v {
			n++
		}
	}
	return n
}

func (s *state) move(delta int) {
	if delta > 0 {
		s.pane(s.focus).SelectNext()
	} else {
		s.pane(s.focus).SelectPrev()
	}
}

func (s *state) page(delta int) {
	if delta > 0 {
		s.pane(s.focus).PageDown()
	} else {
		s.pane(s.focus).PageUp()
	}
}

// descend opens the Org under the cursor, replacing whatever the Repo pane held.
func (s *state) descend() {
	row := s.orgFL.Selected()
	if row == nil || s.mode != modeBrowse {
		return
	}
	s.currentOrg = row.Org
	s.focus = focusRepos
	s.selected = map[string]bool{}
	s.clearQuery(focusRepos)
	s.loadRepos()
}

// enter is Enter and →: open the Org, or start cloning the Selection.
func (s *state) enter() {
	if s.mode != modeBrowse {
		return
	}
	if s.focus == focusOrgs {
		s.descend()
		return
	}
	if s.selectionCount() > 0 {
		s.openClone()
	}
}

// back is Esc and ←: leave the Repo pane (guarding a non-empty Selection, ADR-0005),
// or clear the Org filter.
func (s *state) back() {
	if s.mode != modeBrowse {
		return
	}
	if s.focus == focusOrgs {
		s.clearQuery(focusOrgs)
		return
	}
	if s.selectionCount() > 0 {
		s.setMode(modeLeave)
		return
	}
	s.focus = focusOrgs
	s.sync()
}

// tick ticks or unticks the focused Repo and moves on, fzf-style.
func (s *state) tick(delta int) {
	if s.focus != focusRepos || s.mode != modeBrowse {
		return
	}
	row := s.repoFL.Selected()
	if row == nil {
		return
	}
	row.Ticked = !row.Ticked
	if row.Ticked {
		s.selected[row.Repo.Name] = true
	} else {
		delete(s.selected, row.Repo.Name)
	}
	s.move(delta)
	s.sync()
}

// tickAllMatching ticks every Repo passing the current filter and facets.
func (s *state) tickAllMatching() {
	if s.focus != focusRepos || s.mode != modeBrowse {
		return
	}
	for _, r := range s.repoFL.Filter().Items {
		r.Ticked = true
		s.selected[r.Repo.Name] = true
	}
	s.sync()
}

func (s *state) openClone() {
	// wired to the clone screen in a later commit
	s.setMode(modeClone)
}

// ---- leave prompt ----------------------------------------------------------------

func (s *state) leaveClone() {
	if s.mode != modeLeave {
		return
	}
	s.setMode(modeBrowse)
	s.openClone()
}

func (s *state) leaveDiscard() {
	if s.mode != modeLeave {
		return
	}
	s.selected = map[string]bool{}
	s.rebuildRepos()
	s.focus = focusOrgs
	s.setMode(modeBrowse)
}

func (s *state) leaveStay() {
	if s.mode == modeLeave {
		s.setMode(modeBrowse)
	}
}

// ---- host switch -----------------------------------------------------------------

func (s *state) openHostSwitch() {
	if s.mode != modeBrowse && s.mode != modeFatal && s.mode != modeOptions {
		return
	}
	s.hostCursor = s.hostIdx
	s.setMode(modeHost)
}

func (s *state) moveHost(delta int) {
	s.hostCursor = min(max(s.hostCursor+delta, 0), len(s.hosts)-1)
	s.sync()
}

// confirmHost activates the Host under the cursor and reloads everything from scratch,
// exactly like a fresh launch against it.
func (s *state) confirmHost() {
	if s.mode != modeHost {
		return
	}
	s.hostIdx = s.hostCursor
	s.focus = focusOrgs
	s.currentOrg = forge.Org{}
	s.repoGen++
	s.repoAll, s.reposLoaded, s.reposErr = nil, false, nil
	s.selected = map[string]bool{}
	s.rebuildRepos()
	s.clearQuery(focusOrgs)
	s.clearQuery(focusRepos)
	s.setMode(modeBrowse)
	s.loadOrgs()
}

func (s *state) cancelHost() {
	if s.mode != modeHost {
		return
	}
	if s.fatalErr != nil {
		s.setMode(modeFatal)
		return
	}
	s.setMode(modeBrowse)
}

// ---- derived display state -------------------------------------------------------

func (s *state) sync() {
	s.orgBorder, s.repoBorder = blurredBorder, blurredBorder
	if s.focus == focusOrgs {
		s.orgBorder = focusedBorder
	} else {
		s.repoBorder = focusedBorder
	}
	s.orgW = s.lay.orgWidth
	s.paneH = int16(max(s.height-footerRows, 3))

	s.showBrowse = s.mode != modeFatal && s.mode != modeClone && !s.lay.tooNarrow
	s.showTooNarrow = s.lay.tooNarrow && s.mode != modeFatal && s.mode != modeClone
	s.showFatal = s.mode == modeFatal
	s.showOptions = s.mode == modeOptions
	s.showLeave = s.mode == modeLeave
	s.showHost = s.mode == modeHost
	s.showHelp = s.mode == modeHelp
	s.showClone = s.mode == modeClone

	s.orgTitle = "Orgs"
	s.repoTitle = "Repos"
	if s.currentOrg.Name != "" {
		s.repoTitle = fmt.Sprintf("Repos: %s · %d selected", s.currentOrg.Name, s.selectionCount())
	}

	s.orgChips = "sort: " + s.orgSort.String()
	if s.orgAff != affAll {
		s.orgChips = "affiliation: " + s.orgAff.String() + " · " + s.orgChips
	}
	s.repoChips = fmt.Sprintf("archived: %s · forks: %s · visibility: %s · sort: %s", s.archived, s.fork, s.vis, s.repoSort)

	s.orgMsg = paneMessage(paneInfo{
		what: "orgs", err: s.orgsErr, loaded: s.orgsLoaded, all: len(s.orgAll),
		visible: s.orgFL.Filter().Len(), query: s.query[focusOrgs],
	})
	if s.currentOrg.Name == "" {
		s.repoMsg = "press Enter on an Org to see its Repos"
	} else {
		s.repoMsg = paneMessage(paneInfo{
			what: "repos", err: s.reposErr, loaded: s.reposLoaded, all: len(s.repoAll),
			visible: s.repoFL.Filter().Len(), query: s.query[focusRepos],
		})
	}

	s.footer = fmt.Sprintf(" host: %s · %d selected", s.activeHost().Name, s.selectionCount())
	s.hint = s.keyHints()
	s.status = ""
	if s.transientErr != nil {
		s.status = s.transientErr.Error()
		if d := retryAfter(s.transientErr); d > 0 {
			s.status += fmt.Sprintf(" — retrying in %s", d.Round(time.Second))
		}
	}
	if s.fatalErr != nil {
		s.fatalText = s.fatalErr.Error()
	}
}

type paneInfo struct {
	what         string
	err          error
	loaded       bool
	all, visible int
	query        string
}

// paneMessage distinguishes loading, error, no-matches and empty; never one blank pane.
func paneMessage(p paneInfo) string {
	switch {
	case p.err != nil && p.all == 0:
		return fmt.Sprintf("error loading %s: %v · open options (^o) to reload", p.what, p.err)
	case p.err != nil:
		return fmt.Sprintf("error loading %s: %v · showing what loaded · ^o to reload", p.what, p.err)
	case p.visible > 0:
		return ""
	case !p.loaded && p.all == 0:
		return "loading..."
	case p.query != "" && p.all > 0:
		return fmt.Sprintf("no matches for %q", p.query)
	case p.all > 0:
		return "nothing matches the current options (^o)"
	default:
		return "no " + p.what
	}
}

func (s *state) keyHints() string {
	if s.focus == focusRepos {
		return "type filter · ↑↓ move · tab tick · ^a all · enter clone · esc back · ^o options · F1 help"
	}
	return "type filter · ↑↓ move · enter open · esc clear · ^o options · F1 help · ^c quit"
}
