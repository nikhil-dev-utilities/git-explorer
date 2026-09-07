package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func twoHostFixture() (*fakeForge, []forge.Host) {
	hosts := []forge.Host{{Name: "github.com"}, {Name: "ghe.corp.internal"}}
	f := &fakeForge{
		orgPagesByHost: map[string][]forge.OrgPage{
			"github.com":        {{Orgs: []forge.Org{{Name: "acme"}}}},
			"ghe.corp.internal": {{Orgs: []forge.Org{{Name: "platform"}}}},
		},
	}
	return f, hosts
}

func TestHostSwitch_ListsEveryConfiguredHost(t *testing.T) {
	f, hosts := twoHostFixture()
	tm := newTestModel(t, f, hosts...)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlY})
	m := finalModelAfter(t, tm)

	if m.mode != ModeHostSwitch {
		t.Fatalf("mode = %v, want ModeHostSwitch", m.mode)
	}
	if len(m.hosts) != 2 {
		t.Fatalf("hosts = %+v, want both configured Hosts listed", m.hosts)
	}
	if m.activeHostIdx != 0 {
		t.Errorf("activeHostIdx = %d, want 0 (github.com, unchanged by merely opening the switcher)", m.activeHostIdx)
	}
}

func TestHostSwitch_PickingAHostFetchesFreshOrgsAndReplacesThePane(t *testing.T) {
	f, hosts := twoHostFixture()
	tm := newTestModel(t, f, hosts...)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlY})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // move cursor to ghe.corp.internal
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse after confirming", m.mode)
	}
	if m.activeHostIdx != 1 {
		t.Fatalf("activeHostIdx = %d, want 1 (ghe.corp.internal)", m.activeHostIdx)
	}
	if len(m.orgs) != 1 || m.orgs[0].Name != "platform" {
		t.Fatalf("orgs = %+v, want only platform (the new Host's Orgs)", m.orgs)
	}
	for _, o := range m.orgs {
		if o.Name == "acme" {
			t.Errorf("orgs still contains acme from the previous Host, want it fully replaced")
		}
	}

	var orgsCall int
	for _, h := range f.listOrgsCalls {
		if h.Name == "ghe.corp.internal" {
			orgsCall++
		}
	}
	if orgsCall != 1 {
		t.Errorf("ListOrgs was called %d times for ghe.corp.internal, want exactly 1", orgsCall)
	}
}

func TestHostSwitch_CancelLeavesEverythingUnchanged(t *testing.T) {
	f, hosts := twoHostFixture()
	tm := newTestModel(t, f, hosts...)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlY})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc}) // cancel, not confirm
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse after cancelling", m.mode)
	}
	if m.activeHostIdx != 0 {
		t.Fatalf("activeHostIdx = %d, want still 0 (github.com) — cancel must not switch", m.activeHostIdx)
	}
	if len(m.orgs) != 1 || m.orgs[0].Name != "acme" {
		t.Fatalf("orgs = %+v, want unchanged (still acme from github.com)", m.orgs)
	}
}
