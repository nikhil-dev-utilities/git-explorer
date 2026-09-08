package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case orgsStreamMsg:
		return m.handleOrgsStream(msg)
	case orgPageMsg:
		return m.handleOrgPage(msg)
	case orgsFatalErrMsg:
		return m.handleOrgsFatalErr(msg)
	case repoListMsg:
		return m.handleRepoList(msg)
	case clonePreviewMsg:
		return m.handleClonePreview(msg)
	case cloneRunResultMsg:
		return m.handleCloneRunResult(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.mode {
	case ModeBrowse:
		return m.handleBrowseKey(msg)
	case ModeLeavePrompt:
		return m.handleLeavePromptKeyMsg(msg)
	case ModeHostSwitch:
		return m.handleHostSwitchKeyMsg(msg)
	case ModeFatal:
		return m.handleFatalKeyMsg(msg)
	case ModeCloneDialog:
		return m.handleCloneDialogKeyMsg(msg)
	case ModeCloneRun:
		return m.handleCloneRunKeyMsg(msg)
	case ModeHelp:
		return m.handleHelpKeyMsg(msg)
	}
	return m, nil
}

func (m Model) handleCloneDialogKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m.leaveCloneDialog(), nil
	case tea.KeyTab:
		return m.toggleOrgSubdir()
	case tea.KeyEnter:
		return m.confirmCloneDialog()
	case tea.KeyBackspace:
		return m.editCloneTarget(func(s string) string {
			if len(s) == 0 {
				return s
			}
			return s[:len(s)-1]
		})
	case tea.KeyRunes:
		text := string(msg.Runes)
		return m.editCloneTarget(func(s string) string { return s + text })
	}
	return m, nil
}

// ^c means something different depending on whether a run is actually in flight: it
// cancels the run (the safer read of a stray ^c mid-clone — you can always retry, but
// nuking the whole session is much worse), and only reverts to the ordinary
// quit-the-app binding once the summary is showing.
func (m Model) handleCloneRunKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.cloneRunInFlight {
		if msg.Type == tea.KeyCtrlC {
			return m.cancelCloneRun(), nil
		}
		return m, nil
	}
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m.leaveCloneRunSummary(), nil
	case tea.KeyRunes:
		if len(msg.Runes) == 1 && msg.Runes[0] == 'r' {
			return m.retryFailures()
		}
	}
	return m, nil
}

// From Fatal, nothing works except quitting or switching to a different Host — the
// escape hatch DESIGN.md's failure-surfaces section offers instead of only quitting.
// alt-h is ^y's mnemonic alias here too — it's the same "switch Host" action, on the
// same ctrl key, as Browse's ^y/alt-h.
func (m Model) handleFatalKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlY:
		return m.openHostSwitch(), nil
	case tea.KeyRunes:
		if msg.Alt && len(msg.Runes) == 1 && msg.Runes[0] == 'h' {
			return m.openHostSwitch(), nil
		}
	}
	return m, nil
}

func (m Model) handleHostSwitchKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m.cancelHostSwitch(), nil
	case tea.KeyUp, tea.KeyCtrlP:
		return m.moveHostCursor(-1), nil
	case tea.KeyDown, tea.KeyCtrlN:
		return m.moveHostCursor(1), nil
	case tea.KeyEnter:
		return m.confirmHostSwitch()
	}
	return m, nil
}

// ^c still quits from LeavePrompt — a modal confirmation should never trap the user
// from the one universal escape hatch, even though its "cancel and go back" action is
// Esc (stay), not ^c.
func (m Model) handleLeavePromptKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m.handleLeavePromptEsc(), nil
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			return m.handleLeavePromptKey(msg.Runes[0])
		}
	}
	return m, nil
}

// handleBrowseKey dispatches by key type first (verbs that exist regardless of
// focus), then by focus for anything pane-specific. Every verb here lives on a key
// ADR-0006 permits — see filter.go's cycle helpers and repos.go's descend/backToOrgs
// for what each one does.
//
// An alt+letter combo is checked first: DESIGN.md's mnemonic alt aliases are always
// additive to their ctrl counterpart, never a replacement, so this must never reach
// the ordinary tea.KeyRunes case below (which would otherwise type the letter into
// the focused filter) when the letter is one of the recognized aliases.
func (m Model) handleBrowseKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.Alt && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		if mm, cmd, ok := m.handleBrowseAltKey(msg.Runes[0]); ok {
			return mm, cmd
		}
	}
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyUp, tea.KeyCtrlP:
		return m.moveCursor(-1), nil
	case tea.KeyDown, tea.KeyCtrlN:
		return m.moveCursor(1), nil
	case tea.KeyEnter:
		return m.handleEnter()
	case tea.KeyRight:
		// Additive alias for Enter's Orgs-pane behavior specifically — descend into
		// the highlighted Org. A no-op when already in the Repos pane: Enter still
		// owns "open the clone dialog" there, so Right isn't given a second,
		// different meaning depending on focus the way Enter has.
		if m.focus == FocusOrgs {
			return m.descend()
		}
		return m, nil
	case tea.KeyLeft:
		// Additive alias for Esc's Repos-pane behavior specifically — go back to
		// Orgs, still routed through leaveRepos so a non-empty Selection still gets
		// ADR-0005's LeavePrompt guard. A no-op on the Orgs pane: Esc still owns
		// "clear the filter" there, which Left doesn't take over.
		if m.focus == FocusRepos {
			return m.leaveRepos(), nil
		}
		return m, nil
	case tea.KeyEsc:
		return m.handleEsc(), nil
	case tea.KeyTab:
		return m.toggleSelected(), nil
	case tea.KeyCtrlO:
		return m.selectAllMatching(), nil
	case tea.KeyCtrlT:
		return m.cycleFirstFacet(), nil
	case tea.KeyCtrlF:
		if m.focus == FocusRepos {
			m.forkFilter = nextTriState(m.forkFilter)
		}
		return m, nil
	case tea.KeyCtrlV:
		if m.focus == FocusRepos {
			m.visibility = nextVisibilityFilter(m.visibility)
		}
		return m, nil
	case tea.KeyCtrlS:
		return m.cycleSort(), nil
	case tea.KeyCtrlY:
		return m.openHostSwitch(), nil
	case tea.KeyCtrlR:
		return m.retry()
	case tea.KeyF1:
		return m.openHelp(), nil
	case tea.KeyBackspace:
		return m.editFilter(func(s string) string {
			if len(s) == 0 {
				return s
			}
			return s[:len(s)-1]
		}), nil
	case tea.KeyRunes:
		text := string(msg.Runes)
		return m.editFilter(func(s string) string { return s + text }), nil
	}
	return m, nil
}

// handleBrowseAltKey dispatches Browse mode's alt-key aliases (DESIGN.md's "mnemonic
// alt bindings" bonus, ~line 56). Each case calls exactly the same handler its ctrl
// counterpart calls in the switch below — never a separate implementation that could
// drift — so an alias is always a second path to an existing action, never a new one.
// ok is false for any rune with no alias, telling the caller to fall through to
// ordinary filter-text editing instead.
func (m Model) handleBrowseAltKey(r rune) (Model, tea.Cmd, bool) {
	switch r {
	case 'c': // alt-c: Enter's alias — Orgs: descend · Repos: open clone dialog
		mm, cmd := m.handleEnter()
		return mm, cmd, true
	case 'a': // alt-a: ^o's alias — select all matching
		return m.selectAllMatching(), nil, true
	case 'x': // alt-x: ^t's alias — cycle Affiliation (Orgs) / archived (Repos)
		return m.cycleFirstFacet(), nil, true
	case 'f': // alt-f: ^f's alias — cycle fork (Repos only)
		if m.focus == FocusRepos {
			m.forkFilter = nextTriState(m.forkFilter)
		}
		return m, nil, true
	case 'v': // alt-v: ^v's alias — cycle visibility (Repos only)
		if m.focus == FocusRepos {
			m.visibility = nextVisibilityFilter(m.visibility)
		}
		return m, nil, true
	case 's': // alt-s: ^s's alias — cycle sort
		return m.cycleSort(), nil, true
	case 'h': // alt-h: ^y's alias — switch Host
		return m.openHostSwitch(), nil, true
	case 'r': // alt-r: ^r's alias — retry a pane-scoped load failure
		mm, cmd := m.retry()
		return mm, cmd, true
	}
	return m, nil, false
}

func (m Model) editFilter(edit func(string) string) Model {
	switch m.focus {
	case FocusOrgs:
		m.orgFilter = edit(m.orgFilter)
		m.orgCursor = 0
	case FocusRepos:
		m.repoFilter = edit(m.repoFilter)
		m.repoCursor = 0
	}
	return m
}

func (m Model) moveCursor(delta int) Model {
	switch m.focus {
	case FocusOrgs:
		n := len(filterOrgsByAffiliation(m.orgs, m.orgFilter, m.orgAffiliation))
		m.orgCursor = clampCursor(m.orgCursor+delta, n)
	case FocusRepos:
		n := len(filterRepos(m.repos, m.repoFilter, m.archivedFilter, m.forkFilter, m.visibility))
		m.repoCursor = clampCursor(m.repoCursor+delta, n)
	}
	return m
}

func clampCursor(c, n int) int {
	if n == 0 {
		return 0
	}
	if c < 0 {
		return 0
	}
	if c >= n {
		return n - 1
	}
	return c
}

func (m Model) handleEnter() (Model, tea.Cmd) {
	switch m.focus {
	case FocusOrgs:
		return m.descend()
	case FocusRepos:
		if m.selectionCount() > 0 {
			return m.enterCloneDialog()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleEsc() Model {
	switch m.focus {
	case FocusRepos:
		return m.leaveRepos()
	case FocusOrgs:
		if m.orgFilter != "" {
			m.orgFilter = ""
			m.orgCursor = 0
		}
		return m
	}
	return m
}

// cycleFirstFacet is ^t: the Org pane has only one cyclable facet (Affiliation), so
// it gets the first facet key; the Repo pane's first facet is archived. Reusing one
// key contextually, rather than reserving a second ctrl binding, since DESIGN.md's
// keymap does not name a separate key for the Org pane's Affiliation cycle.
func (m Model) cycleFirstFacet() Model {
	switch m.focus {
	case FocusOrgs:
		m.orgAffiliation = nextAffiliationFilter(m.orgAffiliation)
		m.orgCursor = 0
	case FocusRepos:
		m.archivedFilter = nextTriState(m.archivedFilter)
		m.repoCursor = 0
	}
	return m
}

func (m Model) cycleSort() Model {
	switch m.focus {
	case FocusOrgs:
		m.orgSort = nextSortMode(m.orgSort) // no visible effect — see model.go
	case FocusRepos:
		m.repoSort = nextSortMode(m.repoSort)
	}
	return m
}
