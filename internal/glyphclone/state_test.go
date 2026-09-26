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

func TestShallowToggleReachesEveryRunIncludingRetry(t *testing.T) {
	var got [][]clone.Repo
	run := func(_ context.Context, _ string, repos []clone.Repo, _ bool, _ int, _ func(clone.Event)) []clone.Result {
		got = append(got, repos)
		out := make([]clone.Result, len(repos))
		for i, r := range repos {
			out[i] = clone.Result{Repo: r, Err: errors.New("boom")}
		}
		return out
	}
	repos := []clone.Repo{{Org: "o", Name: "a"}}
	h := newHarness(t, t.TempDir(), repos, run)

	h.s.toggleShallow()
	if !strings.Contains(h.s.options, "shallow: on") {
		t.Errorf("options = %q", h.s.options)
	}
	h.s.confirm()
	h.s.retry()
	if len(got) != 2 || !got[0][0].Shallow || !got[1][0].Shallow {
		t.Errorf("runs = %+v, want both shallow", got)
	}
	if repos[0].Shallow {
		t.Error("confirm mutated the Request's Repos")
	}
	if !h.s.output().Shallow {
		t.Error("Response.Shallow not set")
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

func typeInto(s *state, text string) {
	s.field.Value, s.field.Cursor = text, len(text)
}

func TestGotoPromptPrefillsCurrentDirAndJumpsToTypedPaths(t *testing.T) {
	root := tree(t)
	h := newHarness(t, root, []clone.Repo{{Org: "acme", Name: "r"}}, nil)
	s := h.s

	s.openGoto()
	if !s.prompting || !strings.HasSuffix(s.field.Value, "/") || s.field.Cursor != len(s.field.Value) {
		t.Fatalf("prompt not open/prefilled: %+v", s.field)
	}

	typeInto(s, filepath.Join(root, "alpha", "inner")) // absolute
	s.submitPrompt()
	if s.prompting || s.dir != filepath.Join(root, "alpha", "inner") {
		t.Fatalf("absolute jump failed: prompting=%v dir=%s", s.prompting, s.dir)
	}
	if last := h.target[len(h.target)-1]; last != s.dir {
		t.Errorf("preview not recomputed for jump: %s", last)
	}

	s.openGoto()
	typeInto(s, "../../beta") // relative to current dir
	s.submitPrompt()
	if s.dir != filepath.Join(root, "beta") {
		t.Errorf("relative jump: dir = %s", s.dir)
	}
}

func TestGotoExpandsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home")
	}
	h := newHarness(t, t.TempDir(), nil, nil)
	h.s.openGoto()
	typeInto(h.s, "~")
	h.s.submitPrompt()
	if h.s.dir != home {
		t.Errorf("dir = %s, want %s", h.s.dir, home)
	}
}

func TestPromptRejectsFilesAndKeepsPromptOpen(t *testing.T) {
	root := tree(t)
	h := newHarness(t, root, nil, nil)
	s := h.s

	s.openGoto()
	typeInto(s, filepath.Join(root, "file.txt"))
	s.submitPrompt()
	if !s.prompting || !strings.Contains(s.notice, "is a file") || s.dir != root {
		t.Errorf("file accepted: prompting=%v notice=%q dir=%s", s.prompting, s.notice, s.dir)
	}

	typeInto(s, filepath.Join(root, "file.txt", "below")) // under a file
	s.submitPrompt()
	if !s.prompting || !strings.Contains(s.notice, "cannot use") || s.dir != root {
		t.Errorf("path under a file accepted: prompting=%v notice=%q", s.prompting, s.notice)
	}

	typeInto(s, "  ")
	s.submitPrompt()
	if !s.prompting || s.notice == "" {
		t.Errorf("empty entry accepted")
	}

	s.back() // Esc cancels the prompt, not the screen
	if s.prompting || h.quit != 0 || s.dir != root {
		t.Errorf("esc: prompting=%v quit=%d dir=%s", s.prompting, h.quit, s.dir)
	}
}

func TestNewFolderIsVirtualUntilCloneAndBatchLandsUnderIt(t *testing.T) {
	root := tree(t)
	var calls [][]string
	repos := []clone.Repo{{Org: "acme", Name: "a"}, {Org: "acme", Name: "b"}}
	h := newHarness(t, root, repos, fakeRun(nil, &calls))
	s := h.s

	s.openNew()
	typeInto(s, "clones/acme")
	s.submitPrompt()

	want := filepath.Join(root, "clones", "acme")
	if s.dir != want {
		t.Fatalf("dir = %s, want %s", s.dir, want)
	}
	if !strings.Contains(s.dirNote, "new folder") {
		t.Errorf("dirNote = %q, want the new-folder marker", s.dirNote)
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Errorf("new folder was created before the clone ran: %v", err)
	}
	if got := strings.Join(labels(s), " "); got != "../" {
		t.Errorf("entries = %q, want only ..", got)
	}

	s.confirm()
	for _, r := range s.results {
		if filepath.Dir(r.Dest) != want {
			t.Errorf("%s cloned to %s, want under %s", r.Repo.Name, r.Dest, want)
		}
	}

	// climbing back out of a real dir clears the marker
	s2 := newHarness(t, root, nil, nil).s
	s2.openNew()
	typeInto(s2, "alpha") // exists: just navigates, not "new"
	s2.submitPrompt()
	if s2.dirNote != "" || s2.dir != filepath.Join(root, "alpha") {
		t.Errorf("existing folder marked new: dir=%s note=%q", s2.dir, s2.dirNote)
	}
}

func TestNewFolderRejectsEscapingNames(t *testing.T) {
	h := newHarness(t, tree(t), nil, nil)
	s := h.s
	for _, bad := range []string{"../out", "/abs", ".."} {
		s.openNew()
		typeInto(s, bad)
		s.submitPrompt()
		if !s.prompting || s.notice == "" {
			t.Errorf("%q accepted", bad)
		}
		s.back()
	}
}

func TestBrowsingKeysAreInertWhilePrompting(t *testing.T) {
	root := tree(t)
	h := newHarness(t, root, []clone.Repo{{Org: "o", Name: "a"}}, fakeRun(nil, new([][]string)))
	s := h.s
	s.openGoto()
	s.confirm()
	s.move(1)
	s.open()
	s.toggleOrgSubdir()
	if s.phase != phaseDialog || s.cursor != 0 || s.dir != root || s.orgSubdir {
		t.Errorf("dialog acted while prompting: phase=%v cursor=%d dir=%s sub=%v", s.phase, s.cursor, s.dir, s.orgSubdir)
	}
}
