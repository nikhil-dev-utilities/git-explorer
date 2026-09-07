package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// TestOrgPane_AffiliationFilterCycles covers user story 8's Org-pane half — the
// Affiliation filter — which this slice's issue omitted from its acceptance criteria
// when drafted, even though the parent PRD explicitly asks for it alongside the Repo
// pane's facets. See model.go's AffiliationFilter doc comment.
func TestOrgPane_AffiliationFilterCycles(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{
			{Name: "acme", Affiliation: forge.AffiliationOwner},
			{Name: "globex", Affiliation: forge.AffiliationCollaborator},
			{Name: "initech", Affiliation: forge.AffiliationNone},
		}}},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlT}) // Org-focused: cycles Affiliation, All -> Owner
	m := finalModelAfter(t, tm)

	if m.orgAffiliation != AffiliationFilterOwner {
		t.Fatalf("orgAffiliation = %v, want AffiliationFilterOwner after one ^t", m.orgAffiliation)
	}
	visible := filterOrgsByAffiliation(m.orgs, m.orgFilter, m.orgAffiliation)
	if len(visible) != 1 || visible[0].Name != "acme" {
		t.Fatalf("visible orgs = %+v, want only acme (Affiliation=Owner)", visible)
	}
}
