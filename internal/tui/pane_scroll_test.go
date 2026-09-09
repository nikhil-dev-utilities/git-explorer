package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// This is the actual bug reported live against a real private Host with thousands
// of repos: the cursor kept moving internally, but was almost never inside the
// terminal's real visible window, since nothing windowed the rendered list to the
// pane's bordered-box height. Verified this test fails without the visibleWindow
// fix by temporarily reverting it before writing this comment.

func manyOrgs(n int) []forge.Org {
	orgs := make([]forge.Org, n)
	for i := range orgs {
		orgs[i] = forge.Org{Name: fmt.Sprintf("org%03d", i)}
	}
	return orgs
}

func TestOrgPane_CursorStaysVisibleScrolledDeepIntoALargeList(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: manyOrgs(200)}}}
	tm := newTestModel(t, f) // WithInitialTermSize(80, 24)
	time.Sleep(settleDelay)

	for range 150 {
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	}
	m := finalModelAfter(t, tm)

	if m.orgCursor != 150 {
		t.Fatalf("orgCursor = %d, want 150 (cursor movement itself was never the bug — this confirms the Model side is fine)", m.orgCursor)
	}

	view := m.View()
	if !strings.Contains(view, "> org150") {
		t.Errorf("View() doesn't contain \"> org150\" — the cursor is at index 150 but isn't visibly rendered there. View():\n%s", view)
	}
	if strings.Contains(view, "org000") {
		t.Errorf("View() still contains org000 — the pane isn't actually scrolling away from the top of a 200-item list")
	}
}

func TestRepoPane_CursorStaysVisibleScrolledDeepIntoALargeList(t *testing.T) {
	repos := make([]forge.Repo, 200)
	for i := range repos {
		repos[i] = forge.Repo{Name: fmt.Sprintf("repo%03d", i), Org: "acme"}
	}
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    map[string][]forge.Repo{"acme": repos},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend into Repos
	time.Sleep(settleDelay)

	for range 150 {
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	}
	m := finalModelAfter(t, tm)

	if m.repoCursor != 150 {
		t.Fatalf("repoCursor = %d, want 150", m.repoCursor)
	}

	view := m.View()
	if !strings.Contains(view, "repo150") {
		t.Errorf("View() doesn't contain repo150 — the cursor is at index 150 but isn't visibly rendered there. View():\n%s", view)
	}
	if strings.Contains(view, "repo000") {
		t.Errorf("View() still contains repo000 — the pane isn't actually scrolling away from the top of a 200-item list")
	}
}

// A small list (fewer items than the pane's content height) must render exactly as
// it always did — no windowing artifacts, no missing entries.
func TestOrgPane_SmallListUnaffectedByWindowing(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}, {Name: "globex"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)
	m := finalModelAfter(t, tm)

	view := m.View()
	if !strings.Contains(view, "acme") || !strings.Contains(view, "globex") {
		t.Errorf("View() = %q, want both orgs visible (list fits comfortably within the pane height)", view)
	}
}
