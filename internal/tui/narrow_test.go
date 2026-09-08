package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// newTestModelAtWidth is newTestModel with an explicit terminal width, for the
// layout tests below — otherwise identical (single default Host, same cleanup).
func newTestModelAtWidth(t *testing.T, f *fakeForge, width int) *teatest.TestModel {
	t.Helper()
	m := New(f, []forge.Host{{Name: "github.com"}}, true, noopClonePreview, noopCloneRunner, "", 8)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(width, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
	})
	return tm
}

func narrowRepoFixture() map[string][]forge.Repo {
	return map[string][]forge.Repo{
		"acme": {{
			Name:     "api",
			Org:      "acme",
			Archived: true,
			PushedAt: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		}},
	}
}

// descendedModel drives past Org loading and into the Repo pane for "acme", at the
// given terminal width, returning the settled final Model. The fixture repo is
// Archived, and the archived facet defaults to hide (TriHide) — ^t cycles it to
// show, or the one repo in this fixture would be invisible regardless of width.
func descendedModel(t *testing.T, width int) Model {
	t.Helper()
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    narrowRepoFixture(),
	}
	tm := newTestModelAtWidth(t, f, width)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlT}) // archived: hide -> show
	return finalModelAfter(t, tm)
}

func TestLayout_FullWidthShowsNameBadgesAndDate(t *testing.T) {
	// repoWidth = 120 - orgPaneWidth(28) - paneBorderCols*2(4) - paneGapCols(1) = 87,
	// well above repoPaneFullWidth.
	m := descendedModel(t, 120)
	view := m.View()

	if !strings.Contains(view, "api") {
		t.Fatalf("View() = %q, want the repo name", view)
	}
	if !strings.Contains(view, "archived ") {
		t.Errorf("View() = %q, want the archived badge at full width", view)
	}
	if !strings.Contains(view, "2026-03-15") {
		t.Errorf("View() = %q, want the pushed-at date at full width", view)
	}
}

func TestLayout_MediumWidthDropsDateKeepsBadges(t *testing.T) {
	// repoWidth = 80 - 28 - 4 - 1 = 47, between repoPaneNoDateWidth(46) and
	// repoPaneFullWidth(55).
	m := descendedModel(t, 80)
	view := m.View()

	if !strings.Contains(view, "api") {
		t.Fatalf("View() = %q, want the repo name", view)
	}
	if !strings.Contains(view, "archived ") {
		t.Errorf("View() = %q, want the archived badge still shown", view)
	}
	if strings.Contains(view, "2026-03-15") {
		t.Errorf("View() = %q, want the date dropped at medium width", view)
	}
}

func TestLayout_NarrowWidthNameOnly(t *testing.T) {
	// repoWidth = 65 - 28 - 4 - 1 = 32, below repoPaneNoDateWidth(46), but total width
	// (65) is still >= tooNarrowWidth(60), so the two-pane layout itself survives.
	m := descendedModel(t, 65)
	view := m.View()

	if !strings.Contains(view, "api") {
		t.Fatalf("View() = %q, want the repo name", view)
	}
	if strings.Contains(view, "archived ") {
		t.Errorf("View() = %q, want badges dropped at narrow width", view)
	}
	if strings.Contains(view, "2026-03-15") {
		t.Errorf("View() = %q, want the date dropped at narrow width", view)
	}
}

func TestLayout_BelowTooNarrowWidthShowsSingleMessage(t *testing.T) {
	m := descendedModel(t, 50)
	view := m.View()

	if !strings.Contains(view, "too narrow") {
		t.Fatalf("View() = %q, want the too-narrow message", view)
	}
	if strings.Contains(view, "api") {
		t.Errorf("View() = %q, want no pane content at all below the too-narrow threshold", view)
	}
}
