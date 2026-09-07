package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestHelp_F1OpensAndListsRealBindings(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyF1})
	m := finalModelAfter(t, tm)

	if m.mode != ModeHelp {
		t.Fatalf("mode = %v, want ModeHelp", m.mode)
	}
	view := m.View()
	// A handful of representative entries, spanning different modes, to prove the
	// listing actually reflects keymapTable rather than being empty or static text.
	for _, want := range []string{"select all", "quit", "org-subdirectory", "retry failed"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() = %q, want it to mention %q", view, want)
		}
	}
	// The alt-key column: proves the alias actually renders, not just exists in the
	// table.
	if !strings.Contains(view, "alt-a") {
		t.Errorf("View() = %q, want it to mention the alt-a alias", view)
	}
}

func TestHelp_EscReturnsToBrowse(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyF1})
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse after Esc", m.mode)
	}
}
