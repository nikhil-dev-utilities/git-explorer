package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// Rendered ANSI color codes are stripped by lipgloss when there's no real terminal
// (true of go test's captured output), so these assert against the Style object's
// own getters rather than trying to grep colored escape sequences out of View().

func TestPaneStyle_AppliesTheGivenBorderColor(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	m := finalModelAfter(t, tm)

	style := m.paneStyle(orgPaneWidth, focusedBorderColor)
	if got := style.GetBorderTopForeground(); got != focusedBorderColor {
		t.Errorf("border color = %v, want focusedBorderColor", got)
	}

	// The two colors must actually differ, or focus wouldn't be visually
	// distinguishable at all.
	if focusedBorderColor == blurredBorderColor {
		t.Fatal("focusedBorderColor and blurredBorderColor must not be equal")
	}
}

func TestBrowse_OrgColorAndRepoColorSwapWithFocus(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}

	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	m0 := finalModelAfter(t, tm)
	if m0.focus != FocusOrgs {
		t.Fatalf("focus = %v, want FocusOrgs at startup", m0.focus)
	}
	orgColor0, repoColor0 := m0.paneBorderColors()
	if orgColor0 != focusedBorderColor {
		t.Errorf("orgColor while FocusOrgs = %v, want focusedBorderColor", orgColor0)
	}
	if repoColor0 != blurredBorderColor {
		t.Errorf("repoColor while FocusOrgs = %v, want blurredBorderColor", repoColor0)
	}

	tm2 := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm2.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend into Repos
	m1 := finalModelAfter(t, tm2)
	if m1.focus != FocusRepos {
		t.Fatalf("focus = %v, want FocusRepos after descending", m1.focus)
	}
	orgColor1, repoColor1 := m1.paneBorderColors()
	if orgColor1 != blurredBorderColor {
		t.Errorf("orgColor while FocusRepos = %v, want blurredBorderColor", orgColor1)
	}
	if repoColor1 != focusedBorderColor {
		t.Errorf("repoColor while FocusRepos = %v, want focusedBorderColor", repoColor1)
	}
}

func TestPaneStyle_HeightFillsTerminalShortOfFooter(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f) // WithInitialTermSize(80, 24) — see browse_test.go
	time.Sleep(settleDelay)
	m := finalModelAfter(t, tm)

	style := m.paneStyle(orgPaneWidth, focusedBorderColor)
	// 24 rows total - 2 border rows - 1 footer row (no status line) = 21 content rows.
	if got := style.GetHeight(); got != 21 {
		t.Errorf("content height = %d, want 21 (24 - border(2) - footer(1))", got)
	}
}

func TestPaneStyle_ReservesAnExtraRowWhenStatusLinePresent(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	m := finalModelAfter(t, tm)
	m.transientErr = &forge.Error{Kind: forge.ErrKindTransient, Message: "rate limited"}

	style := m.paneStyle(orgPaneWidth, focusedBorderColor)
	if got := style.GetHeight(); got != 20 {
		t.Errorf("content height = %d, want 20 (24 - border(2) - footer(1) - status(1))", got)
	}
}
