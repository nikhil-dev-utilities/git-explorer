package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// This file doesn't re-run every existing keymap test twice — it exercises one
// representative alt-key alias per mode that has any, asserting each produces the
// identical state transition as its ctrl/named-key counterpart already tested
// elsewhere (selection_test.go's TestSelectAllMatching_*, hosts_test.go's
// TestHostSwitch_ListsEveryConfiguredHost, failures_test.go's Fatal fixture). The
// underlying handler logic is shared (see update.go's handleBrowseAltKey), so this
// is enough to prove the wiring holds without duplicating full coverage.

func altKeyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: true}
}

func TestAltA_SelectAllMatching_SameAsCtrlO(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)

	tm.Type("tf-")
	tm.Send(altKeyMsg('a'))
	m := finalModelAfter(t, tm)

	if !m.selected["tf-network"] || !m.selected["tf-dns"] {
		t.Fatalf("selected = %+v, want tf-network and tf-dns ticked via alt-a", m.selected)
	}
	if m.selected["api"] || m.selected["infra"] {
		t.Errorf("selected = %+v, want api/infra untouched", m.selected)
	}
}

func TestAltH_OpensHostSwitch_SameAsCtrlY(t *testing.T) {
	f, hosts := twoHostFixture()
	tm := newTestModel(t, f, hosts...)
	time.Sleep(settleDelay)

	tm.Send(altKeyMsg('h'))
	m := finalModelAfter(t, tm)

	if m.mode != ModeHostSwitch {
		t.Fatalf("mode = %v, want ModeHostSwitch after alt-h", m.mode)
	}
	if len(m.hosts) != 2 {
		t.Fatalf("hosts = %+v, want both configured Hosts listed", m.hosts)
	}
}

func TestAltC_OpensCloneDialog_SameAsEnter(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // tick "api"
	tm.Send(altKeyMsg('c'))               // alias for Enter: open clone dialog
	m := finalModelAfter(t, tm)

	if m.mode != ModeCloneDialog {
		t.Fatalf("mode = %v, want ModeCloneDialog after alt-c on a non-empty Selection", m.mode)
	}
}

func TestFatalAltH_OpensHostSwitch_SameAsCtrlY(t *testing.T) {
	hosts := []forge.Host{{Name: "github.com"}, {Name: "ghe.corp.internal"}}
	f := &fakeForge{orgsErr: &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated"}}
	tm := newTestModel(t, f, hosts...)
	time.Sleep(settleDelay)

	tm.Send(altKeyMsg('h'))
	m := finalModelAfter(t, tm)

	if m.mode != ModeHostSwitch {
		t.Fatalf("mode = %v, want ModeHostSwitch after alt-h from Fatal", m.mode)
	}
}

// A letter with no alias must still fall through to ordinary filter editing —
// alt-key interception is scoped to the recognized mnemonics only.
func TestAltZ_NoAliasFallsThroughToFilterText(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(altKeyMsg('z'))
	m := finalModelAfter(t, tm)

	if m.orgFilter != "z" {
		t.Errorf("orgFilter = %q, want %q — an unaliased alt+letter should still type into the filter", m.orgFilter, "z")
	}
}
