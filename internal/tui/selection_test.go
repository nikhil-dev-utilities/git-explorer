package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func selectionRepoFixture() map[string][]forge.Repo {
	return map[string][]forge.Repo{
		"acme": {
			{Name: "api", Org: "acme"},
			{Name: "infra", Org: "acme"},
			{Name: "tf-network", Org: "acme"},
			{Name: "tf-dns", Org: "acme"},
		},
	}
}

func TestTab_TicksAndUnticksFocusedRepo(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // tick the first (cursor=0) repo: "api"
	m := finalModelAfter(t, tm)

	if !m.selected["api"] {
		t.Fatalf("selected = %+v, want api ticked", m.selected)
	}
	if m.selectionCount() != 1 {
		t.Errorf("selectionCount() = %d, want 1", m.selectionCount())
	}
}

func TestTab_Untick(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // tick
	tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // untick
	m := finalModelAfter(t, tm)

	if m.selected["api"] {
		t.Errorf("api still ticked after two Tabs, want unticked")
	}
	if m.selectionCount() != 0 {
		t.Errorf("selectionCount() = %d, want 0", m.selectionCount())
	}
}

func TestSelectAllMatching_TicksOnlyFilteredSubset(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Type("tf-")
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlO})
	m := finalModelAfter(t, tm)

	if !m.selected["tf-network"] || !m.selected["tf-dns"] {
		t.Fatalf("selected = %+v, want tf-network and tf-dns ticked", m.selected)
	}
	if m.selected["api"] || m.selected["infra"] {
		t.Errorf("selected = %+v, want api/infra untouched (they never matched the filter)", m.selected)
	}
	if m.selectionCount() != 2 {
		t.Errorf("selectionCount() = %d, want 2", m.selectionCount())
	}
}

func TestSelectAllMatching_LeavesPreviouslyTickedNonMatchingUnchanged(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // hand-tick "api" (cursor starts at 0)
	tm.Type("tf-")
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlO}) // select-all-matching within the filter
	m := finalModelAfter(t, tm)

	if !m.selected["api"] {
		t.Errorf("api was unticked by select-all-matching against an unrelated filter, want it to survive untouched")
	}
	if m.selectionCount() != 3 {
		t.Errorf("selectionCount() = %d, want 3 (api + tf-network + tf-dns)", m.selectionCount())
	}
}

func TestLeavePrompt_EmptySelectionNeverPrompts(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse (empty Selection must never prompt)", m.mode)
	}
	if m.focus != FocusOrgs {
		t.Errorf("focus = %v, want FocusOrgs — navigation should have completed immediately", m.focus)
	}
}

func TestLeavePrompt_NonEmptySelectionPromptsInsteadOfNavigating(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	m := finalModelAfter(t, tm)

	if m.mode != ModeLeavePrompt {
		t.Fatalf("mode = %v, want ModeLeavePrompt", m.mode)
	}
	if m.focus != FocusRepos {
		t.Errorf("focus = %v, want still FocusRepos — navigation must not have completed", m.focus)
	}
	if m.selectionCount() != 1 {
		t.Errorf("selectionCount() = %d, want 1 (unaffected by the prompt itself)", m.selectionCount())
	}
}

func TestLeavePrompt_Discard(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Runes: []rune{'d'}, Type: tea.KeyRunes})
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse after discard", m.mode)
	}
	if m.focus != FocusOrgs {
		t.Errorf("focus = %v, want FocusOrgs — discard completes the navigation", m.focus)
	}
	if m.selectionCount() != 0 {
		t.Errorf("selectionCount() = %d, want 0 after discard", m.selectionCount())
	}
}

func TestLeavePrompt_Stay(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc}) // -> LeavePrompt
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc}) // stay
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse after stay", m.mode)
	}
	if m.focus != FocusRepos {
		t.Errorf("focus = %v, want still FocusRepos — stay cancels the navigation", m.focus)
	}
	if m.selectionCount() != 1 {
		t.Errorf("selectionCount() = %d, want 1 — stay must not clear the Selection", m.selectionCount())
	}
}

func TestLeavePrompt_CloneNowHandsOffSelectionAndMode(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Runes: []rune{'c'}, Type: tea.KeyRunes})
	m := finalModelAfter(t, tm)

	if m.mode != ModeCloneDialog {
		t.Fatalf("mode = %v, want ModeCloneDialog", m.mode)
	}
	if m.selectionCount() != 1 || !m.selected["api"] {
		t.Errorf("selected = %+v, want api still ticked, handed off intact", m.selected)
	}
}
