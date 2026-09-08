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

// When hosts came from composition-root discovery (or its last-resort implicit
// default) rather than the user's own config, a Fatal "not authenticated" failure
// is gentler: pane-scoped with retry, not a full-screen dead end. See
// hostsUserConfigured's doc comment on Model.

func TestDiscoveredHosts_NotAuthenticatedDowngradesToPaneScoped(t *testing.T) {
	f := &fakeForge{orgsErr: &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated for github.com. Run: gh auth login --hostname github.com"}}
	tm := newTestModelWithDiscoveredHosts(t, f)
	m := finalModelAfter(t, tm)

	if m.mode != ModeBrowse {
		t.Fatalf("mode = %v, want ModeBrowse — an undiscovered-auth failure must not take over the screen", m.mode)
	}
	if m.orgsErr == nil {
		t.Fatal("orgsErr = nil, want the downgraded error recorded for inline pane-scoped display")
	}
	view := m.View()
	// A single word, not a phrase — the bordered pane word-wraps at its width, so a
	// multi-word phrase can legitimately split across lines here.
	if !strings.Contains(view, "authenticated") {
		t.Errorf("View() = %q, want the auth message rendered inline", view)
	}
	if !strings.Contains(view, "retry") {
		t.Errorf("View() = %q, want a retry hint", view)
	}
}

func TestDiscoveredHosts_RetryReRunsAuthCheckAgainstTheSameHost(t *testing.T) {
	// Both calls set up front (no runtime mutation of shared fake state, which
	// would race against the fake's own goroutine reading it): the first call's
	// page carries a Fatal Err — equivalent to a synchronous ListOrgs error per
	// forge.OrgPage's own documented contract — simulating the state before `gh
	// auth login`; the second (the retry) simulates having completed it since.
	f := &fakeForge{
		orgPagesSequence: [][]forge.OrgPage{
			{{Err: &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated"}}},
			{{Orgs: []forge.Org{{Name: "acme"}}}},
		},
	}
	tm := newTestModelWithDiscoveredHosts(t, f)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR})
	m := finalModelAfter(t, tm)

	if m.orgsErr != nil {
		t.Errorf("orgsErr = %v, want cleared by a successful retry", m.orgsErr)
	}
	if len(m.orgs) != 1 || m.orgs[0].Name != "acme" {
		t.Errorf("orgs = %+v, want acme from the retry's successful fetch", m.orgs)
	}
}

func TestUserConfiguredHosts_NotAuthenticatedStaysFatal(t *testing.T) {
	// The default newTestModel (hostsUserConfigured: true) already covers this via
	// TestFatal_TakesOverTheWholeScreen — this test exists specifically to name the
	// contrast with the discovered-hosts case above, so the two behaviors are
	// pinned side by side rather than only one of them being obviously tested.
	f := &fakeForge{orgsErr: &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated"}}
	tm := newTestModel(t, f)
	m := finalModelAfter(t, tm)

	if m.mode != ModeFatal {
		t.Fatalf("mode = %v, want ModeFatal — an explicitly user-configured Host's auth failure is a real misconfiguration", m.mode)
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
	loadingModel := New(loadingForge, []forge.Host{{Name: "github.com"}}, true, noopClonePreview, noopCloneRunner, "", 8)
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
