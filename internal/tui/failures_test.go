package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestFatal_TakesOverTheWholeScreen(t *testing.T) {
	f := &fakeForge{orgsErr: &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated, run: gh auth login"}}
	tm := newTestModel(t, f)
	m := finalModelAfter(t, tm)

	if m.mode != ModeFatal {
		t.Fatalf("mode = %v, want ModeFatal", m.mode)
	}
	view := m.View()
	if !strings.Contains(view, "not authenticated") {
		t.Errorf("View() = %q, want the fatal message", view)
	}
}

func TestFatal_HostSwitchEscapeHatchWorks(t *testing.T) {
	hosts := []forge.Host{{Name: "github.com"}, {Name: "ghe.corp.internal"}}
	f := &fakeForge{orgsErr: &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated"}}
	tm := newTestModel(t, f, hosts...)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlY})
	m := finalModelAfter(t, tm)

	if m.mode != ModeHostSwitch {
		t.Fatalf("mode = %v, want ModeHostSwitch — Fatal must offer this escape hatch", m.mode)
	}
}

func TestPaneScoped_PreservesAlreadyLoadedPagesAndOffersRetry(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{
			{Orgs: []forge.Org{{Name: "acme"}}},
			{Err: &forge.Error{Kind: forge.ErrKindPaneScoped, Message: "page 2 failed"}},
		},
	}
	tm := newTestModel(t, f)
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse (pane-scoped never takes over the screen)", m.mode)
	}
	if len(m.orgs) != 1 || m.orgs[0].Name != "acme" {
		t.Fatalf("orgs = %+v, want acme preserved despite the later page failing", m.orgs)
	}
	if m.orgsErr == nil {
		t.Fatal("orgsErr = nil, want the pane-scoped error recorded")
	}
	view := m.View()
	if !strings.Contains(view, "acme") {
		t.Errorf("View() = %q, want acme still rendered", view)
	}
}

func TestPaneScoped_RetryReFetchesCleanly(t *testing.T) {
	f := &fakeForge{
		// First ListOrgs call fails after one page; the retry's call succeeds
		// with a different (but overlapping) set — proving the retry actually
		// re-fetched rather than just clearing the error, and that it replaced
		// rather than appended onto the pre-retry orgs.
		orgPagesSequence: [][]forge.OrgPage{
			{
				{Orgs: []forge.Org{{Name: "acme"}}},
				{Err: &forge.Error{Kind: forge.ErrKindPaneScoped, Message: "page 2 failed"}},
			},
			{
				{Orgs: []forge.Org{{Name: "acme"}, {Name: "globex"}}},
			},
		},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR})
	m := finalModelAfter(t, tm)

	if m.orgsErr != nil {
		t.Errorf("orgsErr = %v, want cleared by a successful retry", m.orgsErr)
	}
	if len(m.orgs) != 2 {
		t.Fatalf("orgs = %+v, want exactly acme+globex from the retry's fetch (not duplicated by appending onto the pre-retry state)", m.orgs)
	}
	if len(f.listOrgsCalls) != 2 {
		t.Errorf("ListOrgs called %d times, want exactly 2 (initial + one retry)", len(f.listOrgsCalls))
	}
}

func TestTransient_RendersInStatusLineOnly(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{
			{Orgs: []forge.Org{{Name: "acme"}}},
			{Err: &forge.Error{Kind: forge.ErrKindTransient, Message: "rate limited", RetryAfter: 45 * time.Second}},
		},
	}
	tm := newTestModel(t, f)
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse — transient must never take over the screen", m.mode)
	}
	if m.transientErr == nil {
		t.Fatal("transientErr = nil, want it recorded")
	}
	if len(m.orgs) != 1 {
		t.Fatalf("orgs = %+v, want acme unaffected by a transient error", m.orgs)
	}
	status := m.statusLine()
	if !strings.Contains(status, "rate limited") || !strings.Contains(status, "45s") {
		t.Errorf("statusLine() = %q, want the message and retry-after duration", status)
	}
	view := m.View()
	if !strings.Contains(view, "acme") {
		t.Errorf("View() = %q, want the pane content still present alongside the status line", view)
	}
}

func TestEmptyStates_AreVisiblyDistinct(t *testing.T) {
	// loading: a freshly constructed Model, before Init/Update ever runs, is
	// exactly the "nothing has arrived yet" state — no need to actually run the
	// program to observe it.
	loadingForge := &fakeForge{orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}}}
	loadingModel := New(loadingForge, []forge.Host{{Name: "github.com"}}, noopClonePreview, noopCloneRunner, "", 8)
	loadingView := loadingModel.View()

	// no orgs at all, load complete.
	emptyForge := &fakeForge{orgPages: []forge.OrgPage{{Orgs: nil}}}
	tm := newTestModel(t, emptyForge)
	emptyModel := finalModelAfter(t, tm)
	emptyView := emptyModel.View()

	// no matches: orgs exist, filter excludes all of them.
	noMatchModel := emptyModel
	noMatchModel.orgs = []forge.Org{{Name: "acme"}}
	noMatchModel.orgFilter = "zzz-nope"
	noMatchModel.orgsLoaded = true
	noMatchView := noMatchModel.View()

	if loadingView == emptyView {
		t.Errorf("loading and empty views are identical: %q", loadingView)
	}
	if loadingView == noMatchView {
		t.Errorf("loading and no-match views are identical: %q", loadingView)
	}
	if emptyView == noMatchView {
		t.Errorf("empty and no-match views are identical: %q", emptyView)
	}
}
