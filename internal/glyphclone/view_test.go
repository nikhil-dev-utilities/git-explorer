package glyphclone

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

func TestDialogRendersBrowserAndPreviewAndKeysDriveIt(t *testing.T) {
	root := tree(t)
	repos := []clone.Repo{{Org: "acme", Name: "tf-network"}}
	h := newHarness(t, root, repos, fakeRun(nil, new([][]string)))

	app := NewApp()
	wire(app, h.s, strings.NewReader(""))

	render := func() string {
		buf := NewBuffer(90, 14)
		app.Template().Execute(buf, 90, 14)
		return buf.StringTrimmed()
	}

	out := render()
	for _, want := range []string{"Clone 1 repo", "Choose target folder", "alpha/", "beta/", "Preview", "tf-network", "org subdirectory: off"} {
		if !strings.Contains(out, want) {
			t.Errorf("dialog missing %q:\n%s", want, out)
		}
	}

	app.Input().Dispatch(riffkey.Key{Special: riffkey.SpecialTab})
	if !h.s.orgSubdir {
		t.Error("<Tab> did not toggle org subdirectory")
	}
	app.Input().Dispatch(riffkey.Key{Special: riffkey.SpecialEscape})
	if h.quit != 1 {
		t.Error("<Esc> did not quit")
	}
}

func TestRunPhaseShowsLogAndSummary(t *testing.T) {
	repos := []clone.Repo{{Org: "acme", Name: "tf-network"}, {Org: "acme", Name: "tf-dns"}}
	h := newHarness(t, t.TempDir(), repos, fakeRun(map[string]bool{"tf-dns": true}, new([][]string)))
	pr, pw := io.Pipe()
	h.s.logw = pw
	app := NewApp()
	wire(app, h.s, pr)

	app.Input().Dispatch(riffkey.Key{Rune: 'c'})

	var out string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		buf := NewBuffer(90, 14)
		app.Template().Execute(buf, 90, 14)
		out = buf.StringTrimmed()
		if strings.Contains(out, "failed: boom line two") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"Clone log", "✓ tf-network", "✗ tf-dns  failed: boom line two", "1 failed", "r retry failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("run view missing %q:\n%s", want, out)
		}
	}
}

func TestPromptRouterCapturesTypingAndEnterJumps(t *testing.T) {
	root := tree(t)
	h := newHarness(t, root, []clone.Repo{{Org: "acme", Name: "r"}}, fakeRun(nil, new([][]string)))
	app := NewApp()
	wire(app, h.s, strings.NewReader(""))

	render := func() string {
		buf := NewBuffer(100, 14)
		app.Template().Execute(buf, 100, 14)
		return buf.StringTrimmed()
	}
	press := func(keys ...riffkey.Key) {
		for _, k := range keys {
			app.Input().Dispatch(k)
		}
	}
	typed := func(s string) {
		for _, r := range s {
			press(riffkey.Key{Rune: r})
		}
	}

	press(riffkey.Key{Rune: '/'})
	if out := render(); !strings.Contains(out, "Go to:") || !strings.Contains(out, "enter go · esc cancel") {
		t.Fatalf("prompt row missing:\n%s", out)
	}

	// 'c' and 'h' are dialog verbs; inside the prompt they must just be text
	press(riffkey.Key{Special: riffkey.SpecialBackspace})
	typed("../ch")
	if h.s.phase != phaseDialog || !strings.HasSuffix(h.s.field.Value, "../ch") {
		t.Fatalf("typing leaked to dialog verbs: phase=%v value=%q", h.s.phase, h.s.field.Value)
	}

	press(riffkey.Key{Special: riffkey.SpecialEscape})
	if h.s.prompting || h.quit != 0 {
		t.Fatalf("esc: prompting=%v quit=%d", h.s.prompting, h.quit)
	}

	// back in the dialog, keys are dialog verbs again
	press(riffkey.Key{Special: riffkey.SpecialTab})
	if !h.s.orgSubdir {
		t.Error("<Tab> did not reach the dialog after the prompt closed")
	}

	press(riffkey.Key{Rune: 'n'})
	typed("fresh2026")
	press(riffkey.Key{Special: riffkey.SpecialEnter})
	out := render()
	if h.s.dir != filepath.Join(root, "fresh2026") || !strings.Contains(out, "new folder, created when cloning") {
		t.Errorf("new folder not shown: dir=%s\n%s", h.s.dir, out)
	}
}
