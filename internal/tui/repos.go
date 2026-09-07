package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// repoListMsg carries the result of ListRepos for the Org that was just selected.
type repoListMsg struct {
	org   forge.Org
	repos []forge.Repo
	err   error
}

// listReposCmd calls the injected Forge's ListRepos, lazily — only ever dispatched in
// direct response to an Org being selected, never eagerly for every Org.
func listReposCmd(f forge.Forge, org forge.Org) tea.Cmd {
	return func() tea.Msg {
		repos, err := f.ListRepos(backgroundCtx(), org)
		return repoListMsg{org: org, repos: repos, err: err}
	}
}

func (m Model) handleRepoList(msg repoListMsg) (Model, tea.Cmd) {
	m.reposLoaded = true
	if msg.err != nil {
		m.reposErr = msg.err
		return m, nil
	}
	m.repos = msg.repos
	return m, nil
}

// descend switches focus to the Repo pane for the Org currently under the cursor in
// the (filtered) Org list, and dispatches a fresh ListRepos call for it — replacing
// whatever the Repo pane previously showed. No caching: re-entering the same Org
// fetches it again, consistent with ADR-0007.
func (m Model) descend() (Model, tea.Cmd) {
	visible := filterOrgs(m.orgs, m.orgFilter)
	if len(visible) == 0 || m.orgCursor < 0 || m.orgCursor >= len(visible) {
		return m, nil
	}
	org := visible[m.orgCursor]

	m.currentOrg = org
	m.focus = FocusRepos
	m.repos = nil
	m.reposLoaded = false
	m.reposErr = nil
	m.repoFilter = ""
	m.repoCursor = 0

	return m, listReposCmd(m.forge, org)
}

// backToOrgs returns focus to the Org pane. The Org pane's own filter and cursor are
// untouched by descending or returning, so they survive automatically.
func (m Model) backToOrgs() Model {
	m.focus = FocusOrgs
	return m
}
