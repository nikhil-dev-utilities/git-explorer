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
			return m.handleLeavePromptKey(msg.Runes[0]), nil
		}
	}
	return m, nil
}

// handleBrowseKey dispatches by key type first (verbs that exist regardless of
// focus), then by focus for anything pane-specific. Every verb here lives on a key
// ADR-0006 permits — see filter.go's cycle helpers and repos.go's descend/backToOrgs
// for what each one does.
func (m Model) handleBrowseKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyUp, tea.KeyCtrlP:
		return m.moveCursor(-1), nil
	case tea.KeyDown, tea.KeyCtrlN:
		return m.moveCursor(1), nil
	case tea.KeyEnter:
		return m.handleEnter()
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
		// Opening the clone dialog on a non-empty Selection is added in a later
		// slice of this PRD (#27 introduces Selection; #31 introduces the dialog).
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
