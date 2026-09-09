package tui

import (
	"strings"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// Regression coverage for issue #87: the Org/Repo pane's quick-filter row used to
// render as a bare, unlabeled line — blank whenever no filter was typed, reported
// live as "the top pane line gets truncated." It must always carry a visible label.

func TestViewOrgPane_FilterLineIsLabeledWhenEmpty(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	m := finalModelAfter(t, tm)

	view := m.View()
	if !strings.Contains(view, "search: (type to filter)") {
		t.Errorf("View() = %q, want the unlabeled-filter placeholder", view)
	}
}

func TestViewOrgPane_FilterLineShowsTheActiveQuery(t *testing.T) {
	f := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	tm := newTestModel(t, f)
	m := finalModelAfter(t, tm)
	m.orgFilter = "acme"

	view := m.View()
	if !strings.Contains(view, "search: acme") {
		t.Errorf("View() = %q, want the query rendered with its label", view)
	}
}
