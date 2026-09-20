package glyphclone

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

// screenHarness mounts an Embedded screen the way the shell does: view as the app's
// view, its router pushed as the modal router.
type screenHarness struct {
	t    *testing.T
	sc   *Embedded
	app  *App
	done []Response
}

func fakePreview(_ context.Context, target string, repos []clone.Repo, sub bool) []clone.Result {
	out := make([]clone.Result, len(repos))
	for i, r := range repos {
		out[i] = clone.Result{Repo: r, Dest: clone.TargetPath(target, r, sub)}
	}
	return out
}

func mountScreen(t *testing.T, target string, repos []clone.Repo, run RunFunc) *screenHarness {
	t.Helper()
	h := &screenHarness{t: t, app: NewApp()}
	sync := func(f func()) { f() }
	h.sc = Embed(Hooks{
		Spawn: sync, Apply: sync, Refresh: func() {},
		Push: h.app.PushRouter, Pop: h.app.PopRouter,
		Done: func(r Response) { h.done = append(h.done, r) },
	})
	h.app.SetView(VBox.Grow(1)(h.sc.View()))
	h.app.PushRouter(h.sc.Router())
	h.sc.Open(Request{Target: target, Repos: repos, Parallelism: 2, Preview: fakePreview, Run: run})
	return h
}

func (h *screenHarness) render(w, ht int) string {
	buf := NewBuffer(w, ht)
	h.app.Template().Execute(buf, int16(w), int16(ht))
	return buf.StringTrimmed()
}

func (h *screenHarness) press(keys ...riffkey.Key) {
	for _, k := range keys {
		h.app.Input().Dispatch(k)
	}
}

func (h *screenHarness) typed(s string) {
	for _, r := range s {
		h.press(riffkey.Key{Rune: r})
	}
}

func special(sp riffkey.Special) riffkey.Key { return riffkey.Key{Special: sp} }

func TestDialogRendersBrowserAndPreviewAndKeysDriveIt(t *testing.T) {
	root := tree(t)
	h := mountScreen(t, root, []clone.Repo{{Org: "acme", Name: "tf-network"}}, fakeRun(nil, new([][]string)))

	out := h.render(90, 14)
	for _, want := range []string{"Clone 1 repo", "Choose target folder", "alpha/", "beta/", "Preview", "tf-network", "org subdirectory: off"} {
		if !strings.Contains(out, want) {
			t.Errorf("dialog missing %q:\n%s", want, out)
		}
	}

	h.press(special(riffkey.SpecialTab))
	if !h.sc.s.orgSubdir {
		t.Error("<Tab> did not toggle org subdirectory")
	}
	h.press(special(riffkey.SpecialEscape))
	if len(h.done) != 1 || h.done[0].Ran || !h.done[0].OrgSubdir {
		t.Errorf("<Esc> should finish the screen with Ran=false and the choices, got %+v", h.done)
	}
}

func TestListKeysMoveTheCursorWithoutTheListsOwnBindings(t *testing.T) {
	root := tree(t)
	h := mountScreen(t, root, nil, nil)
	h.typed("j")
	h.press(special(riffkey.SpecialDown))
	if h.sc.s.cursor != 2 {
		t.Errorf("cursor = %d after j and Down", h.sc.s.cursor)
	}
	h.typed("G")
	if h.sc.s.cursor != len(h.sc.s.entries)-1 {
		t.Errorf("G: cursor = %d", h.sc.s.cursor)
	}
	h.typed("g")
	h.typed("k")
	if h.sc.s.cursor != 0 {
		t.Errorf("g then k: cursor = %d", h.sc.s.cursor)
	}
}

func TestRunPhaseShowsLogAndSummary(t *testing.T) {
	repos := []clone.Repo{{Org: "acme", Name: "tf-network"}, {Org: "acme", Name: "tf-dns"}}
	h := mountScreen(t, t.TempDir(), repos, fakeRun(map[string]bool{"tf-dns": true}, new([][]string)))

	h.typed("c")

	var out string
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if out = h.render(90, 14); strings.Contains(out, "failed: boom line two") {
			break
		}
	}
	for _, want := range []string{"Clone log", "── Clone 2 repos →", "✓ tf-network", "✗ tf-dns  failed: boom line two", "1 failed", "r retry failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("run view missing %q:\n%s", want, out)
		}
	}
}

func TestPromptRouterCapturesTypingAndEnterJumps(t *testing.T) {
	root := tree(t)
	h := mountScreen(t, root, []clone.Repo{{Org: "acme", Name: "r"}}, fakeRun(nil, new([][]string)))
	s := h.sc.s

	h.typed("/")
	if out := h.render(100, 14); !strings.Contains(out, "Go to:") || !strings.Contains(out, "enter go · esc cancel") {
		t.Fatalf("prompt row missing:\n%s", out)
	}

	// 'c' and 'h' are dialog verbs; inside the prompt they must just be text
	h.press(special(riffkey.SpecialBackspace))
	h.typed("../ch")
	if s.phase != phaseDialog || !strings.HasSuffix(s.field.Value, "../ch") {
		t.Fatalf("typing leaked to dialog verbs: phase=%v value=%q", s.phase, s.field.Value)
	}

	h.press(special(riffkey.SpecialEscape))
	if s.prompting || len(h.done) != 0 {
		t.Fatalf("esc: prompting=%v done=%d", s.prompting, len(h.done))
	}

	// back in the dialog, keys are dialog verbs again
	h.press(special(riffkey.SpecialTab))
	if !s.orgSubdir {
		t.Error("<Tab> did not reach the dialog after the prompt closed")
	}

	h.typed("n")
	h.typed("fresh2026")
	h.press(special(riffkey.SpecialEnter))
	out := h.render(100, 14)
	if s.dir != filepath.Join(root, "fresh2026") || !strings.Contains(out, "new folder, created when cloning") {
		t.Errorf("new folder not shown: dir=%s\n%s", s.dir, out)
	}
}

func TestOpenResetsTheScreenForANewRequest(t *testing.T) {
	root := tree(t)
	h := mountScreen(t, root, []clone.Repo{{Org: "acme", Name: "a"}}, fakeRun(nil, new([][]string)))
	h.typed("/")
	h.typed("zzz")
	h.press(special(riffkey.SpecialEnter))
	h.typed("c")
	if h.sc.s.phase == phaseDialog {
		t.Fatal("run did not start")
	}

	h.sc.Open(Request{Target: root, Repos: []clone.Repo{{Org: "acme", Name: "b"}, {Org: "acme", Name: "c"}}, Parallelism: 2, Preview: fakePreview, Run: fakeRun(nil, new([][]string))})
	s := h.sc.s
	if s.phase != phaseDialog || s.showRun || s.prompting || s.dir != root || s.title != "Clone 2 repos" || len(s.results) != 0 || s.ran {
		t.Errorf("stale state after Open: phase=%v showRun=%v prompting=%v dir=%s title=%q results=%d ran=%v",
			s.phase, s.showRun, s.prompting, s.dir, s.title, len(s.results), s.ran)
	}
}
