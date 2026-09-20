package tui

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
	"github.com/nikhil-dev-utilities/git-explorer/internal/glyphclone"
)

// noopCloneScreen is for tests that never open the clone screen.
func noopCloneScreen(glyphclone.Request) (glyphclone.Response, error) {
	return glyphclone.Response{}, nil
}

func cloneModel(screen CloneScreenFunc) Model {
	m := New(&fakeForge{}, []forge.Host{{Name: "github.com"}}, true, screen, "/src", 4)
	m.focus = FocusRepos
	m.repos = []forge.Repo{{Name: "api", Org: "acme"}, {Name: "infra", Org: "acme"}}
	m.selected = map[string]bool{"api": true}
	return m
}

func TestEnterOnSelectionLaunchesCloneScreenWithTheSelection(t *testing.T) {
	m := cloneModel(noopCloneScreen)
	m.cloneOrgSubdir = true

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter with a non-empty Selection returned no command")
	}
	if got.(Model).mode != ModeBrowse {
		t.Errorf("mode = %v, want ModeBrowse (the screen runs outside bubbletea)", got.(Model).mode)
	}

	req := m.cloneRequest()
	if req.Target != "/src" || !req.OrgSubdir || req.Parallelism != 4 {
		t.Errorf("request = %+v", req)
	}
	if len(req.Repos) != 1 || req.Repos[0].Name != "api" || req.Repos[0].Org != "acme" {
		t.Errorf("request repos = %+v, want exactly the ticked repo", req.Repos)
	}
}

func TestEnterWithEmptySelectionDoesNotLaunch(t *testing.T) {
	m := cloneModel(noopCloneScreen)
	m.selected = nil
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Error("Enter with an empty Selection launched the clone screen")
	}
}

func TestCloneScreenDoneAfterARunClearsSelectionAndRemembersChoices(t *testing.T) {
	m := cloneModel(noopCloneScreen)
	got, _ := m.Update(cloneScreenDoneMsg{resp: glyphclone.Response{Ran: true, Target: "/elsewhere", OrgSubdir: true}})
	m = got.(Model)
	if m.selectionCount() != 0 {
		t.Errorf("Selection not cleared after a run: %+v", m.selected)
	}
	if m.cloneTarget != "/elsewhere" || !m.cloneOrgSubdir {
		t.Errorf("target=%q orgSubdir=%v, want them remembered for the next run", m.cloneTarget, m.cloneOrgSubdir)
	}
}

func TestCloneScreenDoneWhenBackedOutKeepsSelection(t *testing.T) {
	m := cloneModel(noopCloneScreen)
	got, _ := m.Update(cloneScreenDoneMsg{resp: glyphclone.Response{Ran: false, Target: "/picked"}})
	m = got.(Model)
	if !m.selected["api"] {
		t.Error("Selection was cleared although nothing was cloned")
	}
	if m.cloneTarget != "/picked" {
		t.Errorf("cloneTarget = %q", m.cloneTarget)
	}
}

func TestCloneScreenErrorChangesNothing(t *testing.T) {
	m := cloneModel(noopCloneScreen)
	got, _ := m.Update(cloneScreenDoneMsg{err: errors.New("no tty"), resp: glyphclone.Response{Ran: true, Target: "/x"}})
	m = got.(Model)
	if !m.selected["api"] || m.cloneTarget != "/src" {
		t.Errorf("error result was applied: selected=%v target=%q", m.selected, m.cloneTarget)
	}
}

// screenSpy returns a CloneScreenFunc that records its Request on a channel and backs
// out immediately, plus a wait func for the test to block on it.
func screenSpy() (CloneScreenFunc, func(*testing.T) glyphclone.Request) {
	ch := make(chan glyphclone.Request, 1)
	screen := func(r glyphclone.Request) (glyphclone.Response, error) {
		ch <- r
		return glyphclone.Response{Target: r.Target}, nil
	}
	return screen, func(t *testing.T) glyphclone.Request {
		t.Helper()
		select {
		case r := <-ch:
			return r
		case <-time.After(3 * time.Second):
			t.Fatal("clone screen was never launched")
			return glyphclone.Request{}
		}
	}
}

// openScreenViaKeys descends into acme, ticks "api", then sends the given keys.
func openScreenViaKeys(t *testing.T, keys ...tea.Msg) glyphclone.Request {
	t.Helper()
	f := &fakeForge{
		orgPages: []forge.OrgPage{{Orgs: []forge.Org{{Name: "acme"}}}},
		repos:    selectionRepoFixture(),
	}
	screen, wait := screenSpy()
	tm := teatest.NewTestModel(t, New(f, []forge.Host{{Name: "github.com"}}, true, screen, "/src", 8),
		teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // descend
	time.Sleep(settleDelay)
	tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // tick "api"
	for _, k := range keys {
		tm.Send(k)
	}
	return wait(t)
}

func TestAltC_LaunchesCloneScreen_SameAsEnter(t *testing.T) {
	req := openScreenViaKeys(t, altKeyMsg('c'))
	if len(req.Repos) != 1 || req.Repos[0].Name != "api" {
		t.Errorf("request repos = %+v, want the ticked repo", req.Repos)
	}
}

func TestLeavePrompt_CloneNowLaunchesCloneScreenWithSelection(t *testing.T) {
	req := openScreenViaKeys(t, tea.KeyMsg{Type: tea.KeyEsc}, tea.KeyMsg{Runes: []rune{'c'}, Type: tea.KeyRunes})
	if len(req.Repos) != 1 || req.Repos[0].Name != "api" {
		t.Errorf("request repos = %+v, want the ticked repo handed off intact", req.Repos)
	}
}
