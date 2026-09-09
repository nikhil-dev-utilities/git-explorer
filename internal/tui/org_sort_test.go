package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// Regression coverage for issue #88: ^s used to update m.orgSort with no visible
// effect at all — the Org pane never sorted by it, and never displayed the current
// sort mode. Org has no activity-like field the way Repo has PushedAt, so it gets
// its own two-state cycle (Name ⇄ Affiliation) rather than Repo's Name ⇄ Activity.

func TestSortOrgs_ByNameIsCaseInsensitive(t *testing.T) {
	orgs := []forge.Org{{Name: "zeta"}, {Name: "Alpha"}, {Name: "beta"}}
	got := sortOrgs(orgs, SortByName)
	want := []string{"Alpha", "beta", "zeta"}
	for i, w := range want {
		if got[i].Name != w {
			t.Fatalf("sortOrgs(SortByName) = %+v, want order %v", got, want)
		}
	}
}

func TestSortOrgs_ByAffiliationGroupsOwnerFirstThenNameWithinGroup(t *testing.T) {
	orgs := []forge.Org{
		{Name: "z-member", Affiliation: forge.AffiliationMember},
		{Name: "a-owner", Affiliation: forge.AffiliationOwner},
		{Name: "b-owner", Affiliation: forge.AffiliationOwner},
		{Name: "none-org", Affiliation: forge.AffiliationNone},
		{Name: "collab-org", Affiliation: forge.AffiliationCollaborator},
	}
	got := sortOrgs(orgs, SortByAffiliation)
	want := []string{"a-owner", "b-owner", "z-member", "collab-org", "none-org"}
	for i, w := range want {
		if got[i].Name != w {
			t.Fatalf("sortOrgs(SortByAffiliation) = %+v, want order %v", got, want)
		}
	}
}

func TestSortOrgs_DoesNotMutateInput(t *testing.T) {
	orgs := []forge.Org{{Name: "zeta"}, {Name: "alpha"}}
	_ = sortOrgs(orgs, SortByName)
	if orgs[0].Name != "zeta" || orgs[1].Name != "alpha" {
		t.Fatalf("sortOrgs mutated its input: %+v", orgs)
	}
}

func TestNextOrgSortMode_CyclesNameAndAffiliationOnly(t *testing.T) {
	if got := nextOrgSortMode(SortByName); got != SortByAffiliation {
		t.Errorf("nextOrgSortMode(SortByName) = %v, want SortByAffiliation", got)
	}
	if got := nextOrgSortMode(SortByAffiliation); got != SortByName {
		t.Errorf("nextOrgSortMode(SortByAffiliation) = %v, want SortByName", got)
	}
}

func orgSortFixture() *fakeForge {
	return &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{
		{Name: "z-member", Affiliation: forge.AffiliationMember},
		{Name: "a-owner", Affiliation: forge.AffiliationOwner},
	}}}}
}

func TestOrgPane_DefaultSortIsNameOrder(t *testing.T) {
	tm := newTestModel(t, orgSortFixture())
	view := finalModelAfter(t, tm).View()

	if !strings.Contains(view, "sort: name") {
		t.Errorf("View() = %q, want the default sort mode shown as \"sort: name\"", view)
	}
	zIdx := strings.Index(view, "z-member")
	aIdx := strings.Index(view, "a-owner")
	if zIdx == -1 || aIdx == -1 || aIdx > zIdx {
		t.Fatalf("View() = %q, want a-owner before z-member under SortByName (alphabetical)", view)
	}
}

// This is the end-to-end reproduction of the reported bug: pressing ^s while the Org
// pane is focused must actually reorder the rendered list and show which sort mode is
// active.
func TestOrgPane_CtrlSCyclesToAffiliationOrder(t *testing.T) {
	tm := newTestModel(t, orgSortFixture())
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlS})
	m := finalModelAfter(t, tm)

	if m.orgSort != SortByAffiliation {
		t.Fatalf("orgSort = %v after ^s, want SortByAffiliation", m.orgSort)
	}

	view := m.View()
	if !strings.Contains(view, "sort: affiliation") {
		t.Errorf("View() = %q, want the sort mode shown as \"sort: affiliation\"", view)
	}
	zIdx := strings.Index(view, "z-member")
	aIdx := strings.Index(view, "a-owner")
	if zIdx == -1 || aIdx == -1 || aIdx > zIdx {
		t.Errorf("View() = %q, want a-owner (Owner) before z-member (Member) under SortByAffiliation", view)
	}
}
