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
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.mode {
	case ModeBrowse:
		return m.handleBrowseKey(msg)
	}
	return m, nil
}

// handleBrowseKey is deliberately minimal in this slice: the filter box is
// always-focused text editing (printable runes append, backspace edits) plus quit.
// Every other verb — navigation, selection, facets, sort, host switch, help — is
// added by later slices of this PRD, each of which extends this switch rather than
// replacing it.
func (m Model) handleBrowseKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyBackspace:
		if len(m.orgFilter) > 0 {
			m.orgFilter = m.orgFilter[:len(m.orgFilter)-1]
		}
		return m, nil
	case tea.KeyRunes:
		m.orgFilter += string(msg.Runes)
		return m, nil
	}
	return m, nil
}
