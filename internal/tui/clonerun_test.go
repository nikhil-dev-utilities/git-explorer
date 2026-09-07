package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// openConfirmedCloneRun descends into "acme", ticks every Repo via select-all, opens
// the clone dialog, and confirms it — leaving the run in flight for a
// fakeCloneRunner, or already resolved for fakeCloneRunnerImmediate, depending on
// which was injected.
func openConfirmedCloneRun(t *testing.T, tm interface{ Send(tea.Msg) }) {
	t.Helper()
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlO}) // select-all-matching
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open clone dialog
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // confirm: start the run
}

// waitForCloneRunnerCall polls fr's call count until the fake has been invoked at
// least once, deterministically synchronizing with the run actually having started
// rather than guessing a sleep duration.
func waitForCloneRunnerCall(fr *fakeCloneRunner) {
	for fr.snapshotCallCount() == 0 {
		time.Sleep(100 * time.Microsecond)
	}
}

func TestCloneRun_CompletesWithMixedOutcomes(t *testing.T) {
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
	runner := fakeCloneRunnerImmediate(map[string]clone.Result{
		"skipped-one":  {Outcome: clone.OutcomeSkipped},
		"conflict-one": {Outcome: clone.OutcomeConflict},
	})
	tm := newTestModelWithCloneRunner(t, f, (&fakeClonePreview{}).fn(), runner, "/src")
	openConfirmedCloneRun(t, tm)
	m := finalModelAfter(t, tm)

	if m.mode != ModeCloneRun {
		t.Fatalf("mode = %v, want ModeCloneRun", m.mode)
	}
	if m.cloneRunInFlight {
		t.Fatal("cloneRunInFlight = true, want false — the fake resolves immediately")
	}
	if len(m.cloneRunResults) != 3 {
		t.Fatalf("cloneRunResults = %+v, want 3", m.cloneRunResults)
	}
	view := m.View()
	if !strings.Contains(view, "cloned: 1") || !strings.Contains(view, "skipped: 1") || !strings.Contains(view, "conflict: 1") {
		t.Errorf("View() = %q, want the three outcomes grouped and counted", view)
	}
}

func TestCloneRun_InFlightShowsIndeterminateState(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    map[string][]forge.Repo{"acme": {{Name: "api", Org: "acme"}}},
	}
	fr := &fakeCloneRunner{}
	tm := newTestModelWithCloneRunner(t, f, (&fakeClonePreview{}).fn(), fr.fn(), "/src")
	openConfirmedCloneRun(t, tm)

	// The run is deliberately still blocked (fakeCloneRunner waits on ctx.Done()).
	// This is the one place in this package where checking the live output stream
	// for a still-in-flight state is reliable: ModeCloneRun's in-flight frame has
	// never been rendered before this point in the test, so there is nothing an
	// earlier read could have already drained.
	waitForOutput(t, tm, "cloning")

	// Release the fake so the test can clean up via t.Cleanup's Quit without
	// leaking a goroutine blocked forever on ctx.Done().
	done := make(chan struct{})
	go func() {
		waitForCloneRunnerCall(fr)
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
		close(done)
	}()
	<-done
}

func TestCloneRun_CancelStopsTheRunAndReturnsToSummary(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    map[string][]forge.Repo{"acme": {{Name: "api", Org: "acme"}}},
	}
	fr := &fakeCloneRunner{}
	tm := newTestModelWithCloneRunner(t, f, (&fakeClonePreview{}).fn(), fr.fn(), "/src")
	openConfirmedCloneRun(t, tm)

	done := make(chan struct{})
	go func() {
		waitForCloneRunnerCall(fr)
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
		close(done)
	}()
	<-done
	m := finalModelAfter(t, tm)

	if m.cloneRunInFlight {
		t.Error("cloneRunInFlight = true, want false — the run must have completed after cancellation")
	}
	if m.mode != ModeCloneRun {
		t.Fatalf("mode = %v, want ModeCloneRun (showing the summary)", m.mode)
	}
	if len(m.cloneRunResults) != 1 {
		t.Fatalf("cloneRunResults = %+v, want 1 (the fake's single repo, completing once cancelled)", m.cloneRunResults)
	}
}

func TestCloneRun_RetryFailuresRedispatchesOnlyTheFailedSubset(t *testing.T) {
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos: map[string][]forge.Repo{
			"acme": {
				{Name: "good", Org: "acme"},
				{Name: "bad", Org: "acme"},
			},
		},
	}
	runner := fakeCloneRunnerImmediate(map[string]clone.Result{
		"bad": {Err: errFake},
	})
	tm := newTestModelWithCloneRunner(t, f, (&fakeClonePreview{}).fn(), runner, "/src")
	openConfirmedCloneRun(t, tm)
	time.Sleep(settleDelay)

	tm.Send(tea.KeyMsg{Runes: []rune{'r'}, Type: tea.KeyRunes})
	m := finalModelAfter(t, tm)

	if len(m.cloneRunResults) != 1 || m.cloneRunResults[0].Repo.Name != "bad" {
		t.Fatalf("cloneRunResults after retry = %+v, want only bad re-attempted", m.cloneRunResults)
	}
}

type fakeErr string

func (e fakeErr) Error() string { return string(e) }

var errFake = fakeErr("simulated failure")
