package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	glyph "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
	"github.com/nikhil-dev-utilities/git-explorer/internal/glyphclone"
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
	modeFilter // typing into the focused pane's filter; every other key is text
)

type focus int

const (
	focusOrgs focus = iota
	focusRepos
)

type cachedRepos struct {
	repos []forge.Repo
	at    time.Time
}

// Deps is everything the UI needs from outside; it never imports a Forge implementation.
type Deps struct {
	Forge               forge.Forge
	Hosts               []forge.Host
	HostsUserConfigured bool
	CloneTarget         string
	Parallelism         int
	Preview             glyphclone.PreviewFunc
	Run                 glyphclone.RunFunc
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
	// repoCache holds each Org's Repos for this session only (ADR-0009): reopening an
	// Org is instant, r/F5 refetches, and the title shows how old the data is.
	repoCache    map[string]cachedRepos
	repoLoadedAt time.Time
	repoFL       *glyph.FilterListC[repoRow]
	selected     map[string]bool

	fatalErr     error
	transientErr error

	// clone screen: target and org-subdirectory carry between runs within a session
	// only (ADR-0007). openScreen is wired to the mounted glyphclone.Screen.
	cloneTarget    string
	cloneOrgSubdir bool
	cloneShallow   bool
	openScreen     func(glyphclone.Request)

	menu       []menuRow
	menuCursor int

	hostRows   []hostRow
	helpRows   []helpRow
	helpCursor int
	helpH      int16
	leaveText  string
	pending    func() // what the leave prompt's "discard" continues with

	// filter text per pane, edited while modeFilter is active
	query [2]string
	cur   [2]int
	text  [2]*riffkey.TextHandler

	width, height int
	orgWidthIdx   int
	lay           layout

	// derived display state, refreshed by sync
	orgBorder, repoBorder glyph.Color
	orgW, paneH           int16
	screenW, screenH      int16
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
		cloneTarget: d.CloneTarget,
		repoCache:   map[string]cachedRepos{},
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
	s.helpRows = helpRows()
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

// handleText edits the focused pane's filter; only meaningful in modeFilter (ADR-0009).
func (s *state) handleText(k riffkey.Key) bool {
	if s.mode != modeFilter {
		return false
	}
	return s.text[s.focus].HandleKey(k)
}

// startFilter enters filter mode for the focused pane.
func (s *state) startFilter() {
	if s.mode == modeBrowse {
		s.setMode(modeFilter)
	}
}

// acceptFilter leaves filter mode keeping the filter applied.
func (s *state) acceptFilter() {
	if s.mode == modeFilter {
		s.setMode(modeBrowse)
	}
}

// cancelFilter clears the focused pane's filter and leaves filter mode.
func (s *state) cancelFilter() {
	if s.mode == modeFilter {
		s.clearQuery(s.focus)
		s.setMode(modeBrowse)
	}
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

func (s *state) cacheKey(o forge.Org) string { return s.activeHost().Name + "/" + o.Name }

// loadRepos fetches the current Org's Repos from the Forge. It runs only on demand:
// the first time an Org is opened this session, or on an explicit reload.
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
			s.repoAll, s.reposLoaded, s.repoLoadedAt = repos, true, s.now()
			s.repoCache[s.cacheKey(org)] = cachedRepos{repos: repos, at: s.repoLoadedAt}
			s.pruneSelection()
			s.rebuildRepos()
		})
	})
}

// pruneSelection drops ticks for Repos that no longer exist after a reload.
func (s *state) pruneSelection() {
	present := make(map[string]bool, len(s.repoAll))
	for _, r := range s.repoAll {
		present[r.Name] = true
	}
	for name := range s.selected {
		if !present[name] {
			delete(s.selected, name)
		}
	}
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

// reload refetches the focused pane from the Forge (r, F5, or the options menu).
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

// descend opens the Org under the cursor. Reopening the Org already shown only moves
// focus: nothing is refetched and the filter and ticks are kept. A different Org first
// passes the selection guard, since opening it discards the current ticks.
func (s *state) descend() {
	row := s.orgFL.Selected()
	if row == nil || s.mode != modeBrowse {
		return
	}
	org := row.Org
	if org.Name == s.currentOrg.Name {
		s.focus = focusRepos
		if s.reposErr != nil {
			s.loadRepos() // retry only after a failure
		}
		s.sync()
		return
	}
	s.guardSelection("opening "+org.Name, func() { s.openOrg(org) })
}

// openOrg shows an Org's Repos, from this session's cache when it has them, otherwise
// by fetching once.
func (s *state) openOrg(org forge.Org) {
	s.currentOrg = org
	s.focus = focusRepos
	s.selected = map[string]bool{}
	s.clearQuery(focusRepos)
	s.repoGen++ // drop any in-flight load for the Org shown before
	s.reposErr, s.transientErr = nil, nil
	if c, ok := s.repoCache[s.cacheKey(org)]; ok {
		s.repoAll, s.reposLoaded, s.repoLoadedAt = c.repos, true, c.at
		s.rebuildRepos()
		return
	}
	s.loadRepos()
}

// cyclePane is Tab and Shift-Tab. It never reloads: with an Org already open it only
// moves focus; with none open yet, Orgs -> Repos opens the highlighted Org.
func (s *state) cyclePane() {
	if s.mode != modeBrowse {
		return
	}
	if s.focus == focusRepos {
		s.focus = focusOrgs
		s.sync()
		return
	}
	if s.currentOrg.Name == "" {
		s.descend()
		return
	}
	s.focus = focusRepos
	s.sync()
}

// enter is Enter: open the Org, or start cloning the Selection.
func (s *state) enter() {
	if s.mode != modeBrowse {
		return
	}
	if s.focus == focusOrgs {
		s.descend()
		return
	}
	s.cloneSelection()
}

// right is → and l: open the highlighted Org. It does nothing in the Repo pane, so a
// stray arrow can never start a clone.
func (s *state) right() {
	if s.mode == modeBrowse && s.focus == focusOrgs {
		s.descend()
	}
}

// left is ← and h: back to the Org pane, keeping everything.
func (s *state) left() {
	if s.mode == modeBrowse && s.focus == focusRepos {
		s.focus = focusOrgs
		s.sync()
	}
}

// back is Esc: clear the focused pane's filter; with none, leave the Repo pane.
func (s *state) back() {
	if s.mode != modeBrowse {
		return
	}
	if s.query[s.focus] != "" {
		s.clearQuery(s.focus)
		return
	}
	s.left()
}

// cloneSelection opens the clone screen for the ticked Repos, if any.
func (s *state) cloneSelection() {
	if s.mode == modeBrowse && s.selectionCount() > 0 {
		s.openClone()
	}
}

// tick ticks or unticks the focused Repo and moves on.
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

// toggleAllMatching ticks every Repo passing the filter and facets; when they are all
// ticked already it unticks them instead.
func (s *state) toggleAllMatching() {
	if s.focus != focusRepos || s.mode != modeBrowse {
		return
	}
	items := s.repoFL.Filter().Items
	allTicked := len(items) > 0
	for _, r := range items {
		if !r.Ticked {
			allTicked = false
			break
		}
	}
	for _, r := range items {
		r.Ticked = !allTicked
		if r.Ticked {
			s.selected[r.Repo.Name] = true
		} else {
			delete(s.selected, r.Repo.Name)
		}
	}
	s.sync()
}

// clearSelection unticks everything, including ticks hidden by the filter or facets.
func (s *state) clearSelection() {
	if s.mode != modeBrowse && s.mode != modeOptions {
		return
	}
	s.selected = map[string]bool{}
	s.rebuildRepos()
}

// selectedCloneRepos converts the Selection into clone.Repo values, resolving each
// CloneURL once through the Forge (internal/clone never computes one itself).
func (s *state) selectedCloneRepos() []clone.Repo {
	var out []clone.Repo
	for _, r := range s.repoAll {
		if s.selected[r.Name] {
			out = append(out, clone.Repo{Org: r.Org, Name: r.Name, CloneURL: s.d.Forge.CloneURL(r)})
		}
	}
	return out
}

func (s *state) openClone() {
	if s.openScreen != nil {
		s.openScreen(glyphclone.Request{
			Target:      s.cloneTarget,
			OrgSubdir:   s.cloneOrgSubdir,
			Shallow:     s.cloneShallow,
			Repos:       s.selectedCloneRepos(),
			Parallelism: s.d.Parallelism,
			Preview:     s.d.Preview,
			Run:         s.d.Run,
		})
	}
	s.setMode(modeClone)
}

// cloneDone is the clone screen returning. The Target, org-subdirectory and shallow choices are
// remembered for the next run; the Selection is cleared only if a run actually
// happened, so backing out leaves it intact.
func (s *state) cloneDone(resp glyphclone.Response) {
	s.cloneTarget, s.cloneOrgSubdir, s.cloneShallow = resp.Target, resp.OrgSubdir, resp.Shallow
	if resp.Ran {
		s.selected = map[string]bool{}
		s.rebuildRepos()
	}
	s.setMode(modeBrowse)
}

// ---- selection guard ------------------------------------------------------------

// guardSelection runs action at once when nothing is ticked. Otherwise it asks first,
// because action (opening another Org, switching Host) would discard the ticks: clone
// them now, discard and continue, or stay (ADR-0005, ADR-0009).
func (s *state) guardSelection(what string, action func()) {
	n := s.selectionCount()
	if n == 0 {
		action()
		return
	}
	noun := "repos"
	if n == 1 {
		noun = "repo"
	}
	s.pending = action
	s.leaveText = fmt.Sprintf("%d %s selected in %s: %s will discard them", n, noun, s.currentOrg.Name, what)
	s.setMode(modeLeave)
}

func (s *state) leaveClone() {
	if s.mode != modeLeave {
		return
	}
	s.pending = nil
	s.setMode(modeBrowse)
	s.openClone()
}

func (s *state) leaveDiscard() {
	if s.mode != modeLeave {
		return
	}
	s.selected = map[string]bool{}
	s.rebuildRepos()
	action := s.pending
	s.pending = nil
	s.setMode(modeBrowse)
	if action != nil {
		action()
	}
}

func (s *state) leaveStay() {
	if s.mode == modeLeave {
		s.pending = nil
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

// confirmHost activates the Host under the cursor, after the selection guard.
func (s *state) confirmHost() {
	if s.mode != modeHost {
		return
	}
	idx := s.hostCursor
	s.guardSelection("switching host", func() { s.switchHost(idx) })
}

// switchHost reloads everything from scratch against Host idx, exactly like a fresh
// launch. The session Repo cache is keyed by Host, so it is not shared across Hosts.
func (s *state) switchHost(idx int) {
	s.hostIdx = idx
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
	s.screenW, s.screenH = int16(s.width), int16(s.height)

	s.showBrowse = s.mode != modeFatal && s.mode != modeClone && !s.lay.tooNarrow
	s.showTooNarrow = s.lay.tooNarrow && s.mode != modeFatal && s.mode != modeClone
	s.showFatal = s.mode == modeFatal
	s.showOptions = s.mode == modeOptions
	s.showLeave = s.mode == modeLeave
	s.showHost = s.mode == modeHost
	s.showHelp = s.mode == modeHelp
	s.showClone = s.mode == modeClone

	s.updateTitles()

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

	s.menu = s.buildMenu()
	s.hostRows = make([]hostRow, len(s.hosts))
	for i, h := range s.hosts {
		s.hostRows[i] = hostRow{Mark: " ", Name: h.Name}
		if i == s.hostIdx {
			s.hostRows[i].Mark = "*"
		}
	}
	s.helpH = int16(min(len(s.helpRows)+4, max(s.height-2, 5)))

	s.footer = fmt.Sprintf(" host: %s · %d selected", s.activeHost().Name, s.selectionCount())
	if s.mode == modeFilter {
		what := "orgs"
		if s.focus == focusRepos {
			what = "repos"
		}
		s.footer += " · filtering " + what
	}
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

// updateTitles refreshes the pane titles. The Repo title carries the age of the data
// (ADR-0007/0009), so it is also refreshed before every frame.
func (s *state) updateTitles() {
	s.orgTitle = "Orgs"
	s.repoTitle = "Repos"
	if s.currentOrg.Name != "" {
		s.repoTitle = fmt.Sprintf("Repos: %s · %d selected", s.currentOrg.Name, s.selectionCount())
		if s.reposLoaded && s.reposErr == nil && !s.repoLoadedAt.IsZero() {
			s.repoTitle += " · loaded " + ago(s.now(), s.repoLoadedAt)
		}
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
		return fmt.Sprintf("error loading %s: %v · press r to reload", p.what, p.err)
	case p.err != nil:
		return fmt.Sprintf("error loading %s: %v · showing what loaded · r to reload", p.what, p.err)
	case p.visible > 0:
		return ""
	case !p.loaded && p.all == 0:
		return "loading..."
	case p.query != "" && p.all > 0:
		return fmt.Sprintf("no matches for %q", p.query)
	case p.all > 0:
		return "nothing matches the current options (o)"
	default:
		return "no " + p.what
	}
}

func (s *state) keyHints() string {
	switch {
	case s.mode == modeFilter:
		return "type to filter · ↑↓ move · enter accept · esc clear · ^c quit"
	case s.focus == focusRepos:
		return "/ filter · space tick · a all · x clear · enter clone · tab orgs · r reload · o options · ? help"
	default:
		return "/ filter · ↑↓ move · enter open · tab repos · r reload · o options · ? help · ^c quit"
	}
}

// ---- options menu ----------------------------------------------------------------

// menuRow is one line of the options menu: a label, its current value, and what
// activating it does (usually cycling the value).
type menuRow struct {
	Label string
	Value string
	act   func()
}

func (s *state) buildMenu() []menuRow {
	return []menuRow{
		{"Host", s.activeHost().Name, s.openHostSwitch},
		{"Orgs: affiliation", s.orgAff.String(), func() { s.orgAff = s.orgAff.next(); s.rebuildOrgs() }},
		{"Orgs: sort", s.orgSort.String(), func() { s.orgSort = otherSort(s.orgSort, sortAffiliation); s.rebuildOrgs() }},
		{"Repos: archived", s.archived.String(), func() { s.archived = s.archived.next(); s.rebuildRepos() }},
		{"Repos: forks", s.fork.String(), func() { s.fork = s.fork.next(); s.rebuildRepos() }},
		{"Repos: visibility", s.vis.String(), func() { s.vis = s.vis.next(); s.rebuildRepos() }},
		{"Repos: sort", s.repoSort.String(), func() { s.repoSort = otherSort(s.repoSort, sortActivity); s.rebuildRepos() }},
		{"Org pane width", fmt.Sprint(orgPaneWidths[s.orgWidthIdx]), s.cycleOrgWidth},
		{"Clear selection", fmt.Sprintf("%d ticked", s.selectionCount()), s.clearSelection},
		{"Reload focused pane", "", func() { s.setMode(modeBrowse); s.reload() }},
	}
}

func (s *state) openOptions() {
	if s.mode == modeBrowse {
		s.setMode(modeOptions)
	}
}

func (s *state) closeOptions() {
	if s.mode == modeOptions {
		s.setMode(modeBrowse)
	}
}

func (s *state) menuMove(delta int) {
	if s.mode == modeOptions {
		s.menuCursor = min(max(s.menuCursor+delta, 0), len(s.menu)-1)
	}
}

func (s *state) menuActivate() {
	if s.mode != modeOptions {
		return
	}
	s.menu[s.menuCursor].act()
	s.sync()
}

// otherSort flips between name and the pane's alternative sort.
func otherSort(cur, alt sortMode) sortMode {
	if cur == sortName {
		return alt
	}
	return sortName
}

type hostRow struct {
	Mark string
	Name string
}

// ---- help ------------------------------------------------------------------------

func (s *state) openHelp() {
	if s.mode == modeBrowse {
		s.helpCursor = 0
		s.setMode(modeHelp)
	}
}

func (s *state) closeHelp() {
	if s.mode == modeHelp {
		s.setMode(modeBrowse)
	}
}

func (s *state) helpMove(delta int) {
	if s.mode == modeHelp {
		s.helpCursor = min(max(s.helpCursor+delta, 0), len(s.helpRows)-1)
	}
}
