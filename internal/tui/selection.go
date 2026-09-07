package tui

import tea "github.com/charmbracelet/bubbletea"

// toggleSelected ticks or unticks the Repo currently under the cursor in the Repo
// pane's filtered list.
func (m Model) toggleSelected() Model {
	if m.focus != FocusRepos {
		return m
	}
	visible := sortRepos(filterRepos(m.repos, m.repoFilter, m.archivedFilter, m.forkFilter, m.visibility), m.repoSort)
	if len(visible) == 0 || m.repoCursor < 0 || m.repoCursor >= len(visible) {
		return m
	}
	name := visible[m.repoCursor].Name
	if m.selected == nil {
		m.selected = make(map[string]bool)
	}
	sel := cloneSelection(m.selected)
	sel[name] = !sel[name]
	m.selected = sel
	return m
}

// selectAllMatching ticks every Repo currently passing the Repo pane's filter,
// leaving non-matching Repos' ticked state unchanged.
func (m Model) selectAllMatching() Model {
	if m.focus != FocusRepos {
		return m
	}
	visible := filterRepos(m.repos, m.repoFilter, m.archivedFilter, m.forkFilter, m.visibility)
	if len(visible) == 0 {
		return m
	}
	sel := cloneSelection(m.selected)
	for _, r := range visible {
		sel[r.Name] = true
	}
	m.selected = sel
	return m
}

func cloneSelection(sel map[string]bool) map[string]bool {
	out := make(map[string]bool, len(sel))
	for k, v := range sel {
		out[k] = v
	}
	return out
}

// leaveRepos is what Esc from the Repo pane routes through: if the Selection is
// non-empty, ADR-0005 requires a LeavePrompt rather than silently discarding or
// silently carrying it; otherwise the navigation completes immediately, exactly as
// before Selection existed.
func (m Model) leaveRepos() Model {
	if m.selectionCount() == 0 {
		return m.backToOrgs()
	}
	m.mode = ModeLeavePrompt
	return m
}

func (m Model) handleLeavePromptKey(r rune) (Model, tea.Cmd) {
	switch r {
	case 'c':
		return m.enterCloneDialog()
	case 'd':
		m.selected = nil
		m.mode = ModeBrowse
		return m.backToOrgs(), nil
	}
	return m, nil
}

func (m Model) handleLeavePromptEsc() Model {
	// Stay: cancel the navigation, return to Browse with focus and Selection
	// exactly as they were.
	m.mode = ModeBrowse
	return m
}
