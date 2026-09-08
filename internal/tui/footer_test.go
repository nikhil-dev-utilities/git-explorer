package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// The persistent footer is a compact status line ("host: ... · N selected") plus a
// nano/mc-style key-hint grid below it, on every frame — not just on error. Browse
// mode had no on-screen hint at all before the status line was first added; the
// grid then replaced a single combined status+hints line that only had room for a
// handful of keys. These tests guard both pieces against silently disappearing.

func TestBrowseFooter_PresentOnFirstFrame(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	m := finalModelAfter(t, tm)

	view := m.View()
	for _, want := range []string{"host:", "github.com", "0 selected", "^y", "switch host", "F1", "help"} {
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
	if !strings.Contains(m0.View(), "descend") {
		t.Errorf("View() = %q, want the Enter hint to say \"descend\" while focused on Orgs", m0.View())
	}

	tm2 := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm2.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend into Repos
	m1 := finalModelAfter(t, tm2)
	if !strings.Contains(m1.View(), "clone") {
		t.Errorf("View() = %q, want the Enter hint to say \"clone\" while focused on Repos", m1.View())
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
