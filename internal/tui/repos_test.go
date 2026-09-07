package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestDescend_EnterOnOrgFetchesItsReposAndSwitchesFocus(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}, {Name: "globex"}}}},
		repos: map[string][]forge.Repo{
			"acme": {{Name: "api", Org: "acme"}, {Name: "infra", Org: "acme"}},
		},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	m := finalModelAfter(t, tm)

	if m.focus != FocusRepos {
		t.Fatalf("focus = %v, want FocusRepos", m.focus)
	}
	if m.currentOrg.Name != "acme" {
		t.Fatalf("currentOrg = %q, want acme (cursor starts at index 0)", m.currentOrg.Name)
	}
	if len(m.repos) != 2 {
		t.Fatalf("got %d repos, want 2: %+v", len(m.repos), m.repos)
	}
	if len(f.listReposCalls) != 1 || f.listReposCalls[0].Name != "acme" {
		t.Fatalf("listReposCalls = %+v, want exactly one call for acme", f.listReposCalls)
	}
}

func TestDescend_OnlyFetchesTheSelectedOrgNotOthers(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}, {Name: "globex"}}}},
		repos: map[string][]forge.Repo{
			"acme":   {{Name: "api", Org: "acme"}},
			"globex": {{Name: "widgets", Org: "globex"}},
		},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // move cursor to globex
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	m := finalModelAfter(t, tm)

	if m.currentOrg.Name != "globex" {
		t.Fatalf("currentOrg = %q, want globex", m.currentOrg.Name)
	}
	if len(f.listReposCalls) != 1 || f.listReposCalls[0].Name != "globex" {
		t.Fatalf("listReposCalls = %+v, want exactly one call for globex only", f.listReposCalls)
	}
}

func TestEsc_FromReposReturnsToOrgsPreservingOrgState(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}, {Name: "acme-labs"}}}},
		repos:    map[string][]forge.Repo{"acme": {{Name: "api", Org: "acme"}}},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Type("acme") // narrow the Org pane before descending
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	m := finalModelAfter(t, tm)

	if m.focus != FocusOrgs {
		t.Fatalf("focus = %v, want FocusOrgs", m.focus)
	}
	if m.orgFilter != "acme" {
		t.Errorf("orgFilter = %q, want it preserved as %q across the round trip", m.orgFilter, "acme")
	}
}

func TestEsc_FromOrgsClearsFilterWhenNonEmpty(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Type("acme")
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	m := finalModelAfter(t, tm)

	if m.orgFilter != "" {
		t.Errorf("orgFilter = %q, want cleared by Esc", m.orgFilter)
	}
	if m.focus != FocusOrgs {
		t.Errorf("focus = %v, want still FocusOrgs (Esc on an empty-filter Org pane does nothing)", m.focus)
	}
}

func reposFixture() map[string][]forge.Repo {
	return map[string][]forge.Repo{
		"acme": {
			{Name: "active", Org: "acme", PushedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			{Name: "old", Org: "acme", PushedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
			{Name: "archived-one", Org: "acme", Archived: true},
		},
	}
}

func TestRepoPane_ArchivedHiddenByDefault(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    reposFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend into acme
	m := finalModelAfter(t, tm)

	visible := filterRepos(m.repos, m.repoFilter, m.archivedFilter, m.forkFilter, m.visibility)
	if len(visible) != 2 {
		t.Fatalf("got %d visible repos with defaults, want 2 (archived hidden): %+v", len(visible), visible)
	}
}

func TestRepoPane_FacetsAndSortToggle(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    reposFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend into acme
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlT}) // cycle archived: hide -> show
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlS}) // sort: name -> activity
	m := finalModelAfter(t, tm)

	if m.archivedFilter != TriShow {
		t.Fatalf("archivedFilter = %v, want TriShow after one ^t", m.archivedFilter)
	}
	visible := sortRepos(filterRepos(m.repos, m.repoFilter, m.archivedFilter, m.forkFilter, m.visibility), m.repoSort)
	if len(visible) != 3 {
		t.Fatalf("got %d visible repos with archived shown, want 3: %+v", len(visible), visible)
	}
	if visible[0].Name != "active" {
		t.Errorf("sorted by activity, most recent first: visible[0] = %q, want active", visible[0].Name)
	}
}
