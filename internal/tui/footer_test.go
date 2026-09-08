package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// The persistent footer is DESIGN.md's own mockup: "host: ... · N selected · ^y
// host · enter clone · F1 help" below the panes, on every frame — not just on error.
// It was missing entirely from Browse mode's View() before this; these tests guard
// against it silently disappearing again.

func TestBrowseFooter_PresentOnFirstFrame(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	m := finalModelAfter(t, tm)

	view := m.View()
	for _, want := range []string{"host:", "github.com", "0 selected", "^y host", "F1 help"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() = %q, want it to contain footer text %q", view, want)
		}
	}
}

func TestBrowseFooter_EnterHintChangesWithFocus(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	m0 := finalModelAfter(t, tm)
	if !strings.Contains(m0.View(), "enter descend") {
		t.Errorf("View() = %q, want \"enter descend\" while focused on Orgs", m0.View())
	}

	tm2 := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm2.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend into Repos
	m1 := finalModelAfter(t, tm2)
	if !strings.Contains(m1.View(), "enter clone") {
		t.Errorf("View() = %q, want \"enter clone\" while focused on Repos", m1.View())
	}
}

func TestBrowseFooter_SelectionCountUpdatesLive(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // tick one repo
	m := finalModelAfter(t, tm)

	if !strings.Contains(m.View(), "1 selected") {
		t.Errorf("View() = %q, want \"1 selected\" after ticking one Repo", m.View())
	}
}

func TestHostSwitchFooter_ShowsKeymapHint(t *testing.T) {
	f, hosts := twoHostFixture()
	tm := newTestModel(t, f, hosts...)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlY})
	m := finalModelAfter(t, tm)

	view := m.View()
	for _, want := range []string{"move", "switch", "cancel", "quit"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() = %q, want it to contain HostSwitch footer text %q", view, want)
		}
	}
}
