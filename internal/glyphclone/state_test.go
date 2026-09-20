package glyphclone

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

type harness struct {
	s      *state
	log    bytes.Buffer
	quit   int
	target []string // targets the preview was asked about, in order
	subs   []bool
	queued []func() // deferred spawns when deferSpawn is set
}

func newHarness(t *testing.T, target string, repos []clone.Repo, run RunFunc) *harness {
	t.Helper()
	h := &harness{}
	preview := func(_ context.Context, target string, repos []clone.Repo, sub bool) []clone.Result {
		h.target = append(h.target, target)
		h.subs = append(h.subs, sub)
		out := make([]clone.Result, len(repos))
		for i, r := range repos {
			out[i] = clone.Result{Repo: r, Dest: clone.TargetPath(target, r, sub)}
		}
		return out
	}
	h.s = newState(Request{Target: target, Repos: repos, Parallelism: 2, Preview: preview, Run: run},
		&h.log, func(f func()) { f() }, func(f func()) { f() }, func() {}, func() { h.quit++ })
	return h
}

func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"alpha/inner", "beta", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func labels(s *state) []string {
	var out []string
	for _, e := range s.entries {
		out = append(out, e.Show)
	}
	return out
}

func TestBrowseListsOnlyVisibleDirsAndNavigates(t *testing.T) {
	root := tree(t)
	h := newHarness(t, root, []clone.Repo{{Org: "acme", Name: "r"}}, nil)
	s := h.s

	if got := strings.Join(labels(s), " "); got != "../ alpha/ beta/" {
		t.Fatalf("entries = %q", got)
	}

	s.move(1) // alpha
	s.open()
	if s.dir != filepath.Join(root, "alpha") || strings.Join(labels(s), " ") != "../ inner/" {
		t.Fatalf("after open: dir=%s entries=%v", s.dir, labels(s))
	}

	s.up()
	if s.dir != root || s.entries[s.cursor].Name != "alpha" {
		t.Errorf("up should return to %s with cursor on alpha, got dir=%s cursor=%d", root, s.dir, s.cursor)
	}

	s.cursor = 0
	s.open() // ".." entry climbs
	if s.dir != filepath.Dir(root) {
		t.Errorf("'..' did not climb: %s", s.dir)
	}
}

func TestMissingTargetListsEmptyButCanClimb(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "not", "yet")
	h := newHarness(t, missing, nil, nil)
	if got := strings.Join(labels(h.s), " "); got != "../" {
		t.Fatalf("entries = %q, want only ..", got)
	}
	h.s.open()
	if h.s.dir != filepath.Join(root, "not") {
		t.Errorf("dir = %s", h.s.dir)
	}
}

func TestPreviewFollowsTargetAndSubdirToggle(t *testing.T) {
	root := tree(t)
	repos := []clone.Repo{{Org: "acme", Name: "r"}}
	h := newHarness(t, root, repos, nil)

	h.s.move(1)
	h.s.open()
	h.s.toggleOrgSubdir()

	last := len(h.target) - 1
	if h.target[last] != filepath.Join(root, "alpha") || !h.subs[last] {
		t.Errorf("last preview = %s sub=%v", h.target[last], h.subs[last])
	}
	want := filepath.Join(root, "alpha", "acme", "r")
	if !strings.Contains(h.s.previewText, filepath.Base(want)) || !strings.Contains(h.s.previewText, "acme") {
		t.Errorf("previewText missing org subdir path:\n%s", h.s.previewText)
	}
	if !strings.Contains(h.s.options, "org subdirectory: on") {
		t.Errorf("options = %q", h.s.options)
	}
}

func TestStalePreviewIsDropped(t *testing.T) {
	var queue []func()
	s := &state{in: Request{Preview: func(_ context.Context, target string, _ []clone.Repo, _ bool) []clone.Result {
		return []clone.Result{{Dest: filepath.Join(target, "m-"+filepath.Base(target))}}
	}}}
	s.spawn = func(f func()) { queue = append(queue, f) }
	s.apply = func(f func()) { f() }
	s.refresh = func() {}
	s.dir = "/first"
	s.requestPreview()
	s.dir = "/second"
	s.requestPreview()
	for _, f := range queue {
		f()
	}
	if !strings.Contains(s.previewText, "m-second") || strings.Contains(s.previewText, "m-first") {
		t.Errorf("stale preview won:\n%s", s.previewText)
	}
}

func fakeRun(fail map[string]bool, calls *[][]string) RunFunc {
	return func(_ context.Context, target string, repos []clone.Repo, sub bool, _ int, on func(clone.Event)) []clone.Result {
		var names []string
		out := make([]clone.Result, len(repos))
		for i, r := range repos {
			names = append(names, r.Name)
			res := clone.Result{Repo: r, Dest: clone.TargetPath(target, r, sub)}
			on(clone.Event{Result: res})
			if fail[r.Name] {
				res.Err = errors.New("boom\nline two")
			}
			out[i] = res
			on(clone.Event{Result: res, Done: true})
		}
		*calls = append(*calls, names)
		return out
	}
}

func TestRunStreamsLogAndRetriesOnlyFailures(t *testing.T) {
	var calls [][]string
	fail := map[string]bool{"b": true}
	repos := []clone.Repo{{Org: "o", Name: "a"}, {Org: "o", Name: "b"}}
	h := newHarness(t, t.TempDir(), repos, fakeRun(fail, &calls))
	s := h.s

	s.confirm()
	if s.phase != phaseDone || s.pct != 100 {
		t.Fatalf("phase=%v pct=%d", s.phase, s.pct)
	}
	log := h.log.String()
	for _, want := range []string{"→ a  cloning...", "✓ a  cloned to", "→ b  cloning...", "✗ b  failed: boom line two"} {
		if !strings.Contains(log, want) {
			t.Errorf("log missing %q:\n%s", want, log)
		}
	}
	if !strings.Contains(s.status, "1 failed") || !strings.Contains(s.keys, "retry") {
		t.Errorf("status=%q keys=%q", s.status, s.keys)
	}

	fail["b"] = false
	s.retry()
	if len(calls) != 2 || strings.Join(calls[1], ",") != "b" {
		t.Fatalf("retry calls = %v, want second call [b] only", calls)
	}
	if len(s.results) != 2 || !strings.Contains(s.status, "0 failed") || strings.Contains(s.keys, "retry") {
		t.Errorf("after retry: results=%d status=%q keys=%q", len(s.results), s.status, s.keys)
	}
}

func TestBackBehaviourPerPhase(t *testing.T) {
	h := newHarness(t, t.TempDir(), nil, nil)
	h.s.back()
	if h.quit != 1 || h.s.output().Ran {
		t.Errorf("dialog back: quit=%d ran=%v", h.quit, h.s.output().Ran)
	}

	h2 := newHarness(t, t.TempDir(), nil, nil)
	cancelled := false
	h2.s.phase = phaseRunning
	h2.s.cancel = func() { cancelled = true }
	h2.s.back()
	if !cancelled || h2.quit != 0 {
		t.Errorf("running back: cancelled=%v quit=%d", cancelled, h2.quit)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home")
	}
	if got := expandHome("~/src"); got != filepath.Join(home, "src") {
		t.Errorf("expandHome = %s", got)
	}
	if got := shortenHome(filepath.Join(home, "src")); got != "~/src" {
		t.Errorf("shortenHome = %s", got)
	}
	if got := expandHome("/abs/~x"); got != "/abs/~x" {
		t.Errorf("expandHome touched non-home path: %s", got)
	}
}

func TestEmptyTargetStartsInCurrentDirectory(t *testing.T) {
	h := newHarness(t, "", nil, nil)
	wd, _ := os.Getwd()
	if h.s.dir != wd {
		t.Errorf("dir = %q, want cwd %q", h.s.dir, wd)
	}
}
