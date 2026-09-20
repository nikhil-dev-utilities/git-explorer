package glyphclone

import (
	"io"
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
