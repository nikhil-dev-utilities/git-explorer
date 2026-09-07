package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func cloneDialogRepoFixture() map[string][]forge.Repo {
	return map[string][]forge.Repo{
		"acme": {
			{Name: "api", Org: "acme"},
			{Name: "infra", Org: "acme"},
		},
	}
}

// newTestModelWithPreview is newTestModel with an explicit ClonePreviewFunc and
// target, and noopCloneRunner (tests that need a real one use
// newTestModelWithCloneRunner instead) — otherwise identical.
func newTestModelWithPreview(t *testing.T, f *fakeForge, preview ClonePreviewFunc, target string) *teatest.TestModel {
	t.Helper()
	return newTestModelWithCloneRunner(t, f, preview, noopCloneRunner, target)
}

func newTestModelWithCloneRunner(t *testing.T, f *fakeForge, preview ClonePreviewFunc, runner CloneRunnerFunc, target string) *teatest.TestModel {
	t.Helper()
	m := New(f, []forge.Host{{Name: "github.com"}}, preview, runner, target, 8)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 24))
	t.Cleanup(func() { _ = tm.Quit() })
	return tm
}

func TestCloneDialog_EnterOnNonEmptySelectionOpensItAndShowsPaths(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    cloneDialogRepoFixture(),
	}
	fp := &fakeClonePreview{}
	tm := newTestModelWithPreview(t, f, fp.fn(), "/src")
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})   // tick "api"
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open dialog
	m := finalModelAfter(t, tm)

	if m.mode != ModeCloneDialog {
		t.Fatalf("mode = %v, want ModeCloneDialog", m.mode)
	}
	if len(m.clonePreviewResults) != 1 || m.clonePreviewResults[0].Repo.Name != "api" {
		t.Fatalf("clonePreviewResults = %+v, want exactly the ticked repo (api)", m.clonePreviewResults)
	}
	wantDest := clone.TargetPath("/src", clone.Repo{Org: "acme", Name: "api"}, false)
	if m.clonePreviewResults[0].Dest != wantDest {
		t.Errorf("Dest = %q, want %q", m.clonePreviewResults[0].Dest, wantDest)
	}
	view := m.View()
	if !containsAll(view, "api", "/src") {
		t.Errorf("View() = %q, want it to show the repo and its destination", view)
	}
}

func TestCloneDialog_EnterWithEmptySelectionDoesNotOpen(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    cloneDialogRepoFixture(),
	}
	fp := &fakeClonePreview{}
	tm := newTestModelWithPreview(t, f, fp.fn(), "/src")
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend, no ticks
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse (empty Selection must not open the dialog)", m.mode)
	}
	if len(fp.calls) != 0 {
		t.Errorf("ClonePreviewFunc called %d times, want 0", len(fp.calls))
	}
}

func TestCloneDialog_ToggleOrgSubdirUpdatesEveryPathLive(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    cloneDialogRepoFixture(),
	}
	fp := &fakeClonePreview{}
	tm := newTestModelWithPreview(t, f, fp.fn(), "/src")
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlO}) // select-all-matching: both repos
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open dialog
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // toggle org-subdirectory on
	m := finalModelAfter(t, tm)

	if !m.cloneOrgSubdir {
		t.Fatalf("cloneOrgSubdir = false, want true after one Tab")
	}
	if len(m.clonePreviewResults) != 2 {
		t.Fatalf("clonePreviewResults = %+v, want both repos", m.clonePreviewResults)
	}
	for _, r := range m.clonePreviewResults {
		want := clone.TargetPath("/src", clone.Repo{Org: "acme", Name: r.Repo.Name}, true)
		if r.Dest != want {
			t.Errorf("repo %s Dest = %q, want %q (org-subdirectory reflected)", r.Repo.Name, r.Dest, want)
		}
	}
	// Two calls: once on opening the dialog, once after the toggle.
	if len(fp.calls) != 2 {
		t.Fatalf("ClonePreviewFunc called %d times, want 2 (open + toggle)", len(fp.calls))
	}
	if !fp.calls[1].orgSubdir {
		t.Errorf("second call's orgSubdir = false, want true")
	}
}

func TestCloneDialog_ShowsAllThreeOutcomes(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos: map[string][]forge.Repo{
			"acme": {
				{Name: "cloned-one", Org: "acme"},
				{Name: "skipped-one", Org: "acme"},
				{Name: "conflict-one", Org: "acme"},
			},
		},
	}
	fp := &fakeClonePreview{byRepo: map[string]clone.Result{
		"skipped-one":  {Outcome: clone.OutcomeSkipped, Dest: "/src/skipped-one"},
		"conflict-one": {Outcome: clone.OutcomeConflict, Dest: "/src/conflict-one"},
	}}
	tm := newTestModelWithPreview(t, f, fp.fn(), "/src")
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlO})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	m := finalModelAfter(t, tm)

	outcomes := map[string]clone.Outcome{}
	for _, r := range m.clonePreviewResults {
		outcomes[r.Repo.Name] = r.Outcome
	}
	if outcomes["cloned-one"] != clone.OutcomeCloned {
		t.Errorf("cloned-one Outcome = %v, want Cloned", outcomes["cloned-one"])
	}
	if outcomes["skipped-one"] != clone.OutcomeSkipped {
		t.Errorf("skipped-one Outcome = %v, want Skipped", outcomes["skipped-one"])
	}
	if outcomes["conflict-one"] != clone.OutcomeConflict {
		t.Errorf("conflict-one Outcome = %v, want Conflict", outcomes["conflict-one"])
	}
	view := m.View()
	if !containsAll(view, "cloned", "skipped", "conflict") {
		t.Errorf("View() = %q, want all three outcome labels to render distinguishably", view)
	}
}

func TestCloneDialog_EscReturnsToBrowseWithSelectionIntact(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    cloneDialogRepoFixture(),
	}
	fp := &fakeClonePreview{}
	tm := newTestModelWithPreview(t, f, fp.fn(), "/src")
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse after Esc", m.mode)
	}
	if m.focus != FocusRepos {
		t.Errorf("focus = %v, want still FocusRepos", m.focus)
	}
	if m.selectionCount() != 1 || !m.selected["api"] {
		t.Errorf("selected = %+v, want api still ticked — Esc must clone nothing and change nothing", m.selected)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
