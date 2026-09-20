package ui

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"

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
