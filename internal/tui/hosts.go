package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// openHostSwitch enters ModeHostSwitch, with the cursor starting on the currently
// active Host.
func (m Model) openHostSwitch() Model {
	m.mode = ModeHostSwitch
	m.hostCursor = m.activeHostIdx
	return m
}

func (m Model) moveHostCursor(delta int) Model {
	m.hostCursor = clampCursor(m.hostCursor+delta, len(m.hosts))
	return m
}

// confirmHostSwitch sets the Host under the cursor active, returns to Browse, and
// triggers a fresh ListOrgs against it — discarding whatever the Org (and, by
// extension, Repo) pane held for the previous Host entirely, not merging or
// appending. No caching, per ADR-0007: switching Hosts is exactly like a fresh
// launch against the new one.
func (m Model) confirmHostSwitch() (Model, tea.Cmd) {
	m.activeHostIdx = m.hostCursor
	m.mode = ModeBrowse

	m.orgs = nil
	m.orgsCh = nil
	m.orgsLoaded = false
	m.orgsErr = nil
	m.orgsFatalErr = nil
	m.orgFilter = ""
	m.orgCursor = 0

	m.focus = FocusOrgs
	m.currentOrg = forge.Org{}
	m.repos = nil
	m.reposLoaded = false
	m.reposErr = nil
	m.repoFilter = ""
	m.repoCursor = 0
	m.selected = nil

	return m, listOrgsCmd(m.forge, m.activeHost())
}

// cancelHostSwitch returns to Browse with the active Host and every pane's contents
// unchanged.
func (m Model) cancelHostSwitch() Model {
	m.mode = ModeBrowse
	return m
}
