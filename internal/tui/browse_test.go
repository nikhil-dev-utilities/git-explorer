package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

const testWaitDuration = 2 * time.Second

// settleDelay gives async tea.Cmd chains (our fake Forge resolves near-instantly)
// time to fully process before we inspect final state. Bubble Tea's own renderer
// diffs frames rather than re-emitting full screen content on every update, which
// makes raw output-byte matching unreliable for anything past the first frame — see
// waitForOutput's doc comment. teatest's own test suite uses the same
// sleep-then-inspect pattern for this reason.
const settleDelay = 200 * time.Millisecond

func newTestModel(t *testing.T, f *fakeForge) *teatest.TestModel {
	t.Helper()
	m := New(f, forge.Host{Name: "github.com"})
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		_ = tm.Quit()
	})
	return tm
}

// waitForOutput proves something appeared in the very first rendered frame — which
// Bubble Tea always renders in full. It is deliberately not used to check content
// after any interaction: subsequent frames are diffed against the previous one, so
// unchanged lines are not re-emitted, and a byte-contains check against them is
// unreliable in both directions (a present-but-unchanged line may not appear in the
// latest read; an absent line's bytes may still be sitting, already read, from an
// earlier frame). State assertions after interaction use finalModelAfter instead.
func waitForOutput(t *testing.T, tm *teatest.TestModel, substr string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(substr))
	}, teatest.WithDuration(testWaitDuration), teatest.WithCheckInterval(10*time.Millisecond))
}

// finalModelAfter waits settleDelay for pending async Cmds to resolve, then quits
// and returns the Model's actual final state — for assertions that need real data or
// a direct call to View() (a plain string, safe to inspect repeatedly, unlike the
// program's live ANSI output stream).
func finalModelAfter(t *testing.T, tm *teatest.TestModel) Model {
	t.Helper()
	time.Sleep(settleDelay)
	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() error = %v", err)
	}
	fm := tm.FinalModel(t, teatest.WithFinalTimeout(testWaitDuration))
	m, ok := fm.(Model)
	if !ok {
		t.Fatalf("FinalModel() = %T, want tui.Model", fm)
	}
	return m
}

func TestBrowse_ShowsLoadingBeforeAnyPageArrives(t *testing.T) {
	gate := make(chan struct{})
	f := &fakeForge{
		orgsGate: gate,
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
	}
	tm := newTestModel(t, f)

	waitForOutput(t, tm, "loading")

	close(gate)
	m := finalModelAfter(t, tm)
	if len(m.orgs) != 1 || m.orgs[0].Name != "acme" {
		t.Fatalf("orgs after gate release = %+v, want [acme]", m.orgs)
	}
}

func TestBrowse_LoadsOrgsProgressively(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{
			{Orgs: []forge.Org{{Name: "acme", Affiliation: forge.AffiliationOwner}}},
			{Orgs: []forge.Org{{Name: "globex", Affiliation: forge.AffiliationMember}}},
		},
	}
	tm := newTestModel(t, f)

	m := finalModelAfter(t, tm)
	if len(m.orgs) != 2 {
		t.Fatalf("got %d orgs in final state, want 2: %+v", len(m.orgs), m.orgs)
	}
	view := m.View()
	if !strings.Contains(view, "acme") || !strings.Contains(view, "globex") {
		t.Errorf("View() = %q, want it to contain both org names", view)
	}
}

func TestBrowse_SubstringFilterNarrowsList(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{
			{Name: "acme"},
			{Name: "acme-labs"},
			{Name: "globex"},
		}}},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay) // let all three orgs load before we start typing

	tm.Type("acme")
	m := finalModelAfter(t, tm)

	if m.orgFilter != "acme" {
		t.Fatalf("orgFilter = %q, want acme", m.orgFilter)
	}
	visible := orgNames(filterOrgs(m.orgs, m.orgFilter))
	want := []string{"acme", "acme-labs"}
	if len(visible) != len(want) {
		t.Fatalf("visible orgs = %v, want %v (globex must be excluded)", visible, want)
	}
	for i := range want {
		if visible[i] != want[i] {
			t.Errorf("visible[%d] = %q, want %q", i, visible[i], want[i])
		}
	}

	view := m.View()
	if !strings.Contains(view, "acme-labs") {
		t.Errorf("View() = %q, want it to contain acme-labs", view)
	}
	if strings.Contains(view, "globex") {
		t.Errorf("View() = %q, want globex excluded after filtering to acme", view)
	}
}

func TestBrowse_RegexFilter(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{
			{Name: "tf-network"},
			{Name: "tf-dns"},
			{Name: "docs"},
		}}},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Type("/^tf-")
	m := finalModelAfter(t, tm)

	if m.orgFilter != "/^tf-" {
		t.Fatalf("orgFilter = %q, want /^tf-", m.orgFilter)
	}
	visible := orgNames(filterOrgs(m.orgs, m.orgFilter))
	want := []string{"tf-network", "tf-dns"}
	if len(visible) != len(want) {
		t.Fatalf("visible orgs = %v, want %v (docs must be excluded)", visible, want)
	}

	view := m.View()
	if strings.Contains(view, "docs") {
		t.Errorf("View() = %q, want docs excluded by ^tf-", view)
	}
}

func TestBrowse_BackspaceEditsFilter(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}, {Name: "globex"}}}},
	}
	tm := newTestModel(t, f)
	time.Sleep(settleDelay)

	tm.Type("acmex")
	tm.Send(tea.KeyMsg{Type: tea.KeyBackspace})
	m := finalModelAfter(t, tm)

	if m.orgFilter != "acme" {
		t.Errorf("orgFilter = %q, want %q after one backspace on %q", m.orgFilter, "acme", "acmex")
	}
}
