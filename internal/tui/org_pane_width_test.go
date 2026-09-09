package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestNew_DefaultsToTheLargerOrgPaneWidthPreset(t *testing.T) {
	m := New(&fakeForge{}, []forge.Host{{Name: "github.com"}}, true, noopClonePreview, noopCloneRunner, "", 8)
	if m.orgPaneWidthIdx != defaultOrgPaneWidthIdx {
		t.Fatalf("orgPaneWidthIdx = %d, want defaultOrgPaneWidthIdx (%d)", m.orgPaneWidthIdx, defaultOrgPaneWidthIdx)
	}
	if got := orgPaneWidthPresets[m.orgPaneWidthIdx]; got <= orgPaneWidthPresets[0] {
		t.Fatalf("default preset = %d, want it larger than the smallest preset (%d)", got, orgPaneWidthPresets[0])
	}
}

func TestCycleOrgPaneWidth_WrapsAroundThePresetList(t *testing.T) {
	m := New(&fakeForge{}, []forge.Host{{Name: "github.com"}}, true, noopClonePreview, noopCloneRunner, "", 8)
	seen := map[int]bool{m.orgPaneWidthIdx: true}
	for range len(orgPaneWidthPresets) {
		m = m.cycleOrgPaneWidth()
		seen[m.orgPaneWidthIdx] = true
	}
	if len(seen) != len(orgPaneWidthPresets) {
		t.Fatalf("cycled through %d distinct indices, want all %d presets visited", len(seen), len(orgPaneWidthPresets))
	}
	if m.orgPaneWidthIdx != defaultOrgPaneWidthIdx {
		t.Fatalf("orgPaneWidthIdx after a full cycle = %d, want back to defaultOrgPaneWidthIdx (%d)", m.orgPaneWidthIdx, defaultOrgPaneWidthIdx)
	}
}

// This is the end-to-end reproduction: a real ^g keypress must actually change what
// View() renders, not just update state nothing reads.
func TestOrgPane_CtrlGWidensTheRenderedPane(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlG})
	after := finalModelAfter(t, tm)

	if after.orgPaneWidthIdx != defaultOrgPaneWidthIdx+1 {
		t.Fatalf("orgPaneWidthIdx after one ^g = %d, want %d (default + 1)", after.orgPaneWidthIdx, defaultOrgPaneWidthIdx+1)
	}
	wantWidth := orgPaneWidthPresets[defaultOrgPaneWidthIdx+1]
	if got := after.orgPaneWidth(); got != wantWidth {
		t.Errorf("rendered Org pane width after ^g = %d, want %d", got, wantWidth)
	}
}

// TestOrgPaneWidth_ClampsToLeaveTheRepoPaneUsable confirms a wide preset chosen on a
// wide terminal doesn't crush the Repo pane if the terminal is then narrow (or was
// already narrow) — see orgPaneWidth's own doc comment.
func TestOrgPaneWidth_ClampsToLeaveTheRepoPaneUsable(t *testing.T) {
	m := New(&fakeForge{}, []forge.Host{{Name: "github.com"}}, true, noopClonePreview, noopCloneRunner, "", 8)
	m.width = tooNarrowWidth // the narrowest width viewBrowse still renders two panes at
	for m.orgPaneWidthIdx != len(orgPaneWidthPresets)-1 {
		m = m.cycleOrgPaneWidth()
	}

	got := m.orgPaneWidth()
	repoWidth := m.width - got - paneGapCols - paneBorderCols*2
	if repoWidth < minRepoPaneWidth {
		t.Fatalf("repoWidth = %d at the widest preset and narrowest supported terminal, want >= minRepoPaneWidth (%d)", repoWidth, minRepoPaneWidth)
	}
}
