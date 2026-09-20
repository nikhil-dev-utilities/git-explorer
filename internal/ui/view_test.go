package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

type harness struct {
	t   *testing.T
	s   *state
	app *App
}

func mount(t *testing.T, f *fakeForge, userConfigured bool) *harness {
	t.Helper()
	d := Deps{
		Forge:               f,
		Hosts:               []forge.Host{{Name: "github.com"}, {Name: "ghe.corp"}},
		HostsUserConfigured: userConfigured,
		Parallelism:         4,
		CloneTarget:         t.TempDir(),
		Preview:             fakePreview,
		Run:                 fakeRun,
	}
	sync := func(fn func()) { fn() }
	s := newState(d, sync, sync, func() {})
	app := NewApp()
	wire(app, s)
	return &harness{t: t, s: s, app: app}
}

// render draws one frame at w x h. Glyph list navigation reads row counts recorded
// during rendering, so a frame must be drawn before keys that move the cursor.
func (h *harness) render(w, ht int) string {
	h.t.Helper()
	h.s.resize(w, ht)
	buf := NewBuffer(w, ht)
	h.app.Template().Execute(buf, int16(w), int16(ht))
	return buf.StringTrimmed()
}

func (h *harness) press(specials ...riffkey.Special) {
	for _, sp := range specials {
		h.app.Input().Dispatch(riffkey.Key{Special: sp})
	}
}

func (h *harness) ctrl(r rune) { h.app.Input().Dispatch(riffkey.Key{Rune: r, Mod: riffkey.ModCtrl}) }

func (h *harness) typed(text string) {
	for _, r := range text {
		h.app.Input().Dispatch(riffkey.Key{Rune: r})
	}
}

func TestBrowseRendersBothPanesWithBadgesAndAge(t *testing.T) {
	f := defaultFake()
	f.repos["acme"][0].PushedAt = testNow().Add(-3 * 24 * 3600 * 1e9)
	h := mount(t, f, true)
	h.s.now = testNow
	h.s.enter()
	out := h.render(140, 16)
	for _, want := range []string{"Orgs", "Repos: acme · 0 selected", "acme", "globex", "tf-network", "3d ago",
		"archived: hide", "host: github.com", "type filter"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestTypingFiltersTheFocusedPaneOnly(t *testing.T) {
	h := mount(t, defaultFake(), true)
	h.render(100, 14)

	h.typed("gl")
	out := h.render(100, 14)
	if !strings.Contains(out, "globex") || strings.Contains(out, "platform-eng") {
		t.Errorf("Org filter not applied:\n%s", out)
	}

	h.press(riffkey.SpecialEnter) // descend into globex (no repos), focus repos
	h.typed("zz")
	if h.s.query[focusOrgs] != "gl" || h.s.query[focusRepos] != "zz" {
		t.Errorf("queries = %q", h.s.query)
	}
	h.press(riffkey.SpecialEscape)
	if h.s.focus != focusOrgs {
		t.Errorf("Esc did not return focus to Orgs")
	}
}

func TestKeysMoveTickAndAdvance(t *testing.T) {
	h := mount(t, defaultFake(), true)
	h.render(120, 16)
	h.press(riffkey.SpecialEnter) // acme
	h.render(120, 16)

	h.press(riffkey.SpecialTab, riffkey.SpecialTab)
	if h.s.selectionCount() != 2 || !h.s.selected["tf-dns"] || !h.s.selected["tf-network"] {
		t.Fatalf("selected = %v", h.s.selected)
	}
	if got := h.s.repoFL.Selected(); got == nil || got.Repo.Name != "web" {
		t.Errorf("cursor = %+v, want web after two ticks", got)
	}
	out := h.render(120, 16)
	if !strings.Contains(out, "2 selected") || !strings.Contains(out, "[x]") {
		t.Errorf("ticks not shown:\n%s", out)
	}

	h.press(riffkey.SpecialUp)
	h.ctrl('a') // tick all matching
	if h.s.selectionCount() != 3 {
		t.Errorf("^a: selected = %v", h.s.selected)
	}
	h.press(riffkey.SpecialEscape) // selection non-empty -> leave prompt
	if h.s.mode != modeLeave {
		t.Errorf("mode = %v, want leave prompt", h.s.mode)
	}
}

func TestPanesReflowAndShedColumnsOnResize(t *testing.T) {
	f := defaultFake()
	f.repos["acme"][0].PushedAt = testNow().Add(-3 * 24 * 3600 * 1e9)
	h := mount(t, f, true)
	h.s.now = testNow
	h.s.enter()

	wide := h.render(140, 14)
	mid := h.render(90, 14)
	narrow := h.render(64, 14)
	tiny := h.render(50, 14)

	if !strings.Contains(wide, "3d ago") {
		t.Errorf("wide lost the age column:\n%s", wide)
	}
	if strings.Contains(mid, "3d ago") {
		t.Errorf("mid should shed the age column:\n%s", mid)
	}
	if !strings.Contains(narrow, "tf-network") || strings.Contains(narrow, "3d ago") {
		t.Errorf("narrow should keep names, drop the rest:\n%s", narrow)
	}
	if !strings.Contains(tiny, "terminal too narrow") {
		t.Errorf("below 60 columns should show only the message:\n%s", tiny)
	}
	// every frame fits its width: no line wider than the terminal
	for w, out := range map[int]string{140: wide, 90: mid, 64: narrow} {
		for _, line := range strings.Split(out, "\n") {
			if n := len([]rune(line)); n > w {
				t.Errorf("width %d: line is %d wide: %q", w, n, line)
			}
		}
	}
}

func TestLongOrgListScrollsWithTheCursorAndFitsPaneHeight(t *testing.T) {
	var orgs []forge.Org
	for i := 0; i < 3000; i++ {
		orgs = append(orgs, org(fmt.Sprintf("org-%04d", i), forge.AffiliationMember))
	}
	f := &fakeForge{pages: map[string][]forge.OrgPage{"github.com": {{Orgs: orgs}}}}
	h := mount(t, f, true)
	h.render(100, 20)
	for i := 0; i < 60; i++ {
		h.press(riffkey.SpecialDown)
		h.render(100, 20)
	}
	out := h.render(100, 20)
	if !strings.Contains(out, "org-0060") {
		t.Errorf("cursor row scrolled out of view:\n%s", out)
	}
	if lines := strings.Count(out, "\n") + 1; lines > 20 {
		t.Errorf("frame is %d lines tall in a 20 line terminal", lines)
	}
}

func TestFatalShowsTheFixCommandOverEmptyPanes(t *testing.T) {
	f := &fakeForge{orgsErr: &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated: run gh auth login"}}
	h := mount(t, f, true)
	out := h.render(100, 14)
	if !strings.Contains(out, "gh auth login") || !strings.Contains(out, "^y switch host") {
		t.Errorf("fatal view:\n%s", out)
	}
}

func TestShortTerminalKeepsBordersAndFooterIntact(t *testing.T) {
	h := mount(t, defaultFake(), true)
	h.s.enter()
	for _, ht := range []int{9, 10, 12, 16} {
		out := h.render(100, ht)
		lines := strings.Split(out, "\n")
		if len(lines) > ht {
			t.Errorf("height %d: frame is %d lines", ht, len(lines))
		}
		if !strings.Contains(out, "type filter") || !strings.Contains(out, "host: github.com") {
			t.Errorf("height %d: footer overwritten:\n%s", ht, out)
		}
		bottoms := 0
		for _, l := range lines {
			if strings.Contains(l, "╰") {
				bottoms++
				if strings.Count(l, "╰") != 2 || strings.Count(l, "╯") != 2 {
					t.Errorf("height %d: pane bottoms are on different rows:\n%s", ht, out)
				}
			}
		}
		if bottoms != 1 {
			t.Errorf("height %d: %d bottom-border rows:\n%s", ht, bottoms, out)
		}
	}
}

func TestOptionsMenuCyclesFacetsAndOwnsTheKeysWhileOpen(t *testing.T) {
	h := mount(t, defaultFake(), true)
	h.render(110, 22)
	h.press(riffkey.SpecialEnter) // acme
	h.render(110, 22)

	h.ctrl('o')
	out := h.render(110, 22)
	for _, want := range []string{"Options", "Host", "github.com", "Repos: archived", "hide", "Org pane width", "Reload focused pane"} {
		if !strings.Contains(out, want) {
			t.Errorf("options card missing %q:\n%s", want, out)
		}
	}

	h.typed("xyz") // browse typing must be inert under the menu
	if h.s.query[focusRepos] != "" {
		t.Errorf("typing leaked to the filter under the menu: %q", h.s.query)
	}

	h.press(riffkey.SpecialDown, riffkey.SpecialDown, riffkey.SpecialDown) // -> Repos: archived
	h.press(riffkey.SpecialEnter)
	if h.s.archived != triShow || len(repoNames(h.s)) != 4 {
		t.Errorf("archived=%v repos=%v, want archived shown (4 repos)", h.s.archived, repoNames(h.s))
	}
	if !strings.Contains(h.render(110, 22), "archived: show") {
		t.Error("chip line did not follow the facet")
	}

	h.press(riffkey.SpecialEscape)
	if h.s.mode != modeBrowse {
		t.Fatalf("mode = %v after Esc", h.s.mode)
	}
	h.typed("tf")
	if h.s.query[focusRepos] != "tf" {
		t.Errorf("filter typing did not resume after closing the menu: %q", h.s.query)
	}
}

func TestOptionsRowsCoverEveryFacetAndReload(t *testing.T) {
	f := defaultFake()
	s := newTest(t, f, true)
	s.enter()

	s.openOptions()
	labels := []string{}
	for _, r := range s.menu {
		labels = append(labels, r.Label)
	}
	want := "Host,Orgs: affiliation,Orgs: sort,Repos: archived,Repos: forks,Repos: visibility,Repos: sort,Org pane width,Reload focused pane"
	if join(labels) != want {
		t.Fatalf("rows = %s", join(labels))
	}

	activate := func(label string) {
		for i, r := range s.menu {
			if r.Label == label {
				s.menuCursor = i
			}
		}
		s.menuActivate()
	}
	activate("Orgs: affiliation")
	if s.orgAff != affOwner || join(orgNames(s)) != "acme" {
		t.Errorf("affiliation owner: %v %v", s.orgAff, orgNames(s))
	}
	activate("Orgs: sort")
	if s.orgSort != sortAffiliation {
		t.Errorf("orgSort = %v", s.orgSort)
	}
	activate("Repos: forks")
	activate("Repos: visibility")
	activate("Repos: sort")
	if s.fork != triShow || s.vis != visPublic || s.repoSort != sortActivity {
		t.Errorf("fork=%v vis=%v sort=%v", s.fork, s.vis, s.repoSort)
	}
	activate("Org pane width")
	if s.orgWidthIdx != 2 {
		t.Errorf("orgWidthIdx = %d", s.orgWidthIdx)
	}

	calls := len(f.repoCalls)
	activate("Reload focused pane")
	if s.mode != modeBrowse || len(f.repoCalls) != calls+1 {
		t.Errorf("reload: mode=%v repoCalls=%d->%d", s.mode, calls, len(f.repoCalls))
	}

	s.openOptions()
	activate("Host")
	if s.mode != modeHost {
		t.Errorf("Host row did not open the host switcher: %v", s.mode)
	}
}

func TestLeavePromptCardAndItsKeys(t *testing.T) {
	h := mount(t, defaultFake(), true)
	h.render(110, 20)
	h.press(riffkey.SpecialEnter)
	h.render(110, 20)
	h.press(riffkey.SpecialTab)
	h.press(riffkey.SpecialEscape)

	out := h.render(110, 20)
	if h.s.mode != modeLeave || !strings.Contains(out, "1 repo selected in acme") || !strings.Contains(out, "c clone now") {
		t.Fatalf("leave card missing (mode %v):\n%s", h.s.mode, out)
	}
	h.typed("x") // inert
	if h.s.query[focusRepos] != "" {
		t.Error("typing leaked under the leave prompt")
	}
	h.typed("d")
	if h.s.mode != modeBrowse || h.s.selectionCount() != 0 || h.s.focus != focusOrgs {
		t.Errorf("d: mode=%v sel=%d focus=%v", h.s.mode, h.s.selectionCount(), h.s.focus)
	}
}

func TestHostSwitchThroughOptionsReloadsFromTheNewHost(t *testing.T) {
	f := defaultFake()
	h := mount(t, f, true)
	h.render(110, 22)

	h.ctrl('o')
	h.press(riffkey.SpecialEnter) // Host row
	out := h.render(110, 22)
	if h.s.mode != modeHost || !strings.Contains(out, "Hosts") || !strings.Contains(out, "* github.com") || !strings.Contains(out, "ghe.corp") {
		t.Fatalf("host card (mode %v):\n%s", h.s.mode, out)
	}
	h.press(riffkey.SpecialDown, riffkey.SpecialEnter)
	if h.s.mode != modeBrowse || h.s.activeHost().Name != "ghe.corp" || join(orgNames(h.s)) != "corp" {
		t.Errorf("mode=%v host=%s orgs=%v", h.s.mode, h.s.activeHost().Name, orgNames(h.s))
	}
	if !strings.Contains(h.render(110, 22), "host: ghe.corp") {
		t.Error("footer did not switch host")
	}
	h.typed("co") // browse keys are back
	if h.s.query[focusOrgs] != "co" {
		t.Errorf("typing did not resume: %q", h.s.query)
	}
}

func TestHelpOverlayListsTheKeymapAndScrolls(t *testing.T) {
	h := mount(t, defaultFake(), true)
	h.render(120, 40)
	h.press(riffkey.SpecialF1)
	out := h.render(120, 40)
	for _, want := range []string{"Help", "Browse", "tick every Repo matching the filter", "LeavePrompt", "fzf:"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q:\n%s", want, out)
		}
	}
	h.press(riffkey.SpecialDown, riffkey.SpecialDown)
	if h.s.helpCursor != 2 {
		t.Errorf("helpCursor = %d", h.s.helpCursor)
	}
	h.press(riffkey.SpecialEscape)
	if h.s.mode != modeBrowse {
		t.Errorf("mode = %v", h.s.mode)
	}
	if got := h.render(120, 8); strings.Count(got, "\n")+1 > 8 {
		t.Errorf("help overflows an 8 row terminal:\n%s", got)
	}
}

func TestFatalOffersHostSwitchAndReturnsToFatalOnCancel(t *testing.T) {
	f := defaultFake()
	f.errByHost = map[string]error{"github.com": &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated: run gh auth login"}}
	h := mount(t, f, true)
	if h.s.mode != modeFatal {
		t.Fatalf("mode = %v", h.s.mode)
	}
	h.typed("zz") // inert under the fatal card
	if h.s.query[focusOrgs] != "" {
		t.Error("typing leaked under the fatal card")
	}

	h.ctrl('y')
	if h.s.mode != modeHost {
		t.Fatalf("^y: mode = %v", h.s.mode)
	}
	h.press(riffkey.SpecialEscape)
	if h.s.mode != modeFatal {
		t.Errorf("cancel should return to the fatal card, got %v", h.s.mode)
	}

	h.ctrl('y')
	h.press(riffkey.SpecialDown, riffkey.SpecialEnter)
	if h.s.mode != modeBrowse || join(orgNames(h.s)) != "corp" || h.s.fatalErr != nil {
		t.Errorf("switch away from the broken host: mode=%v orgs=%v fatal=%v", h.s.mode, orgNames(h.s), h.s.fatalErr)
	}
}

func fakePreview(_ context.Context, target string, repos []clone.Repo, sub bool) []clone.Result {
	out := make([]clone.Result, len(repos))
	for i, r := range repos {
		out[i] = clone.Result{Repo: r, Dest: clone.TargetPath(target, r, sub)}
	}
	return out
}

func fakeRun(_ context.Context, target string, repos []clone.Repo, sub bool, _ int, on func(clone.Event)) []clone.Result {
	out := make([]clone.Result, len(repos))
	for i, r := range repos {
		res := clone.Result{Repo: r, Dest: clone.TargetPath(target, r, sub)}
		on(clone.Event{Result: res})
		on(clone.Event{Result: res, Done: true})
		out[i] = res
	}
	return out
}

// pollFor re-renders until want appears (the Log widget consumes its pipe asynchronously).
func (h *harness) pollFor(w, ht int, want string) string {
	h.t.Helper()
	var out string
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if out = h.render(w, ht); strings.Contains(out, want) {
			return out
		}
	}
	h.t.Fatalf("never saw %q:\n%s", want, out)
	return out
}

func (h *harness) selectTwoAndOpenClone() {
	h.render(120, 30)
	h.press(riffkey.SpecialEnter) // acme
	h.render(120, 30)
	h.press(riffkey.SpecialTab, riffkey.SpecialTab)
	h.press(riffkey.SpecialEnter) // clone
}

func TestCloneScreenRunsInTheSharedAppAndClearsTheSelection(t *testing.T) {
	h := mount(t, defaultFake(), true)
	h.selectTwoAndOpenClone()

	out := h.render(120, 30)
	if h.s.mode != modeClone || !strings.Contains(out, "Clone 2 repos") || !strings.Contains(out, "Choose target folder") ||
		!strings.Contains(out, "tf-dns") || strings.Contains(out, "Repos: acme") {
		t.Fatalf("clone screen not shown full-screen (mode %v):\n%s", h.s.mode, out)
	}
	h.typed("x") // no browse typing under the clone screen
	if h.s.query[focusRepos] != "" {
		t.Error("typing leaked to the repo filter under the clone screen")
	}

	h.typed("c")
	out = h.pollFor(120, 30, "done: 2 cloned")
	for _, want := range []string{"Clone log", "✓ tf-dns", "✓ tf-network", "esc done"} {
		if !strings.Contains(out, want) {
			t.Errorf("run view missing %q:\n%s", want, out)
		}
	}

	h.press(riffkey.SpecialEscape)
	out = h.render(120, 30)
	if h.s.mode != modeBrowse || h.s.selectionCount() != 0 || !strings.Contains(out, "0 selected") {
		t.Errorf("after the run: mode=%v selected=%d\n%s", h.s.mode, h.s.selectionCount(), out)
	}
	h.typed("we") // browse keys are live again
	if h.s.query[focusRepos] != "we" {
		t.Errorf("filter typing did not resume: %q", h.s.query)
	}
}

func TestBackingOutOfTheCloneScreenKeepsSelectionAndRemembersTarget(t *testing.T) {
	h := mount(t, defaultFake(), true)
	h.selectTwoAndOpenClone()
	h.render(120, 30)

	h.typed("/")
	h.typed("elsewhere")
	target := h.s.cloneTarget
	h.press(riffkey.SpecialEnter) // jump to the typed (new) folder
	h.press(riffkey.SpecialTab)   // org subdirectory on
	h.press(riffkey.SpecialEscape)

	if h.s.mode != modeBrowse || h.s.selectionCount() != 2 {
		t.Fatalf("mode=%v selected=%d, want browse with the Selection intact", h.s.mode, h.s.selectionCount())
	}
	if !h.s.cloneOrgSubdir || !strings.HasSuffix(h.s.cloneTarget, "elsewhere") || h.s.cloneTarget == target {
		t.Errorf("choices not remembered: target=%q sub=%v", h.s.cloneTarget, h.s.cloneOrgSubdir)
	}

	h.press(riffkey.SpecialEnter) // reopen: starts from what was chosen
	out := h.render(120, 30)
	if !strings.Contains(out, "org subdirectory: on") || !strings.Contains(out, "new folder, created when cloning") {
		t.Errorf("reopened screen did not start from the remembered choices:\n%s", out)
	}
}
