package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// Right/Left are additive aliases for Enter's/Esc's Orgs<->Repos navigation
// specifically — never a second meaning layered onto Enter/Esc themselves, and never
// active on the pane they don't apply to (Right on Repos, Left on Orgs are no-ops,
// leaving Enter/Esc's own behavior there untouched).

func TestRight_OnOrgsDescendsSameAsEnter(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	m := finalModelAfter(t, tm)

	if m.focus != FocusRepos {
		t.Fatalf("focus = %v, want FocusRepos after Right on Orgs", m.focus)
	}
	if m.currentOrg.Name != "acme" {
		t.Fatalf("currentOrg = %q, want acme", m.currentOrg.Name)
	}
}

func TestRight_OnReposIsANoOp(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend into Repos
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	m := finalModelAfter(t, tm)

	if m.focus != FocusRepos {
		t.Fatalf("focus = %v, want still FocusRepos — Right must be a no-op here", m.focus)
	}
	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want still ModeBrowse", m.mode)
	}
}

func TestLeft_OnReposReturnsToOrgsSameAsEsc(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyLeft})
	m := finalModelAfter(t, tm)

	if m.focus != FocusOrgs {
		t.Fatalf("focus = %v, want FocusOrgs after Left on Repos", m.focus)
	}
}

func TestLeft_OnReposWithSelectionPromptsSameAsEsc(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // tick a repo — non-empty Selection

	tm.Send(tea.KeyMsg{Type: tea.KeyLeft})
	m := finalModelAfter(t, tm)

	if m.mode != ModeLeavePrompt {
		t.Fatalf("mode = %v, want ModeLeavePrompt — Left must still respect ADR-0005's guard", m.mode)
	}
	if m.focus != FocusRepos {
		t.Fatalf("focus = %v, want still FocusRepos — navigation must not have completed", m.focus)
	}
}

func TestLeft_OnOrgsIsANoOp(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyLeft})
	m := finalModelAfter(t, tm)

	if m.focus != FocusOrgs {
		t.Fatalf("focus = %v, want still FocusOrgs", m.focus)
	}
	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want still ModeBrowse — Left must be a no-op here", m.mode)
	}
}
