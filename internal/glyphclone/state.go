// Package glyphclone is git-explorer's clone screen: a folder browser that picks the
// Target with a live pre-flight preview, then a streaming deploy-log of the Clone Run.
// It is mounted inside the shell's Glyph app (see Embed).
//
// state.go holds the behaviour with no terminal dependency, so it is tested directly.
package glyphclone

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	glyph "github.com/kungfusheep/glyph"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

// PreviewFunc classifies repos against target without cloning anything.
type PreviewFunc func(ctx context.Context, target string, repos []clone.Repo, orgSubdir bool) []clone.Result

// RunFunc executes a Clone Run, reporting per-Repo progress through onEvent.
// clone.RunProgress has exactly this signature.
type RunFunc func(ctx context.Context, target string, repos []clone.Repo, orgSubdir bool, parallelism int, onEvent func(clone.Event)) []clone.Result

// Request is everything the clone screen needs; it never touches a Forge or config.
type Request struct {
	Target      string
	OrgSubdir   bool
	Shallow     bool
	Repos       []clone.Repo
	Parallelism int
	Preview     PreviewFunc
	Run         RunFunc
}

// Response is what the shell needs back. Ran is false when the user backed out, in which
// case the Selection should be left alone.
type Response struct {
	Ran       bool
	Target    string
	OrgSubdir bool
	Shallow   bool
	Results   []clone.Result
}

type phase int

const (
	phaseDialog phase = iota
	phaseRunning
	phaseDone
)

type promptKind int

const (
	promptGoto promptKind = iota + 1
	promptNew
)

const (
	dialogHints = "enter open · ← up · / go to path · n new folder · tab org subdirectory · s shallow · c clone · esc cancel"
	promptHints = "enter go · esc cancel"
)

type entry struct {
	Name string
	Up   bool
	Show string
}

func newEntry(name string, up bool) entry {
	if up {
		return entry{Up: true, Show: "../"}
	}
	return entry{Name: name, Show: name + "/"}
}

// state is mutated only from the render goroutine (key handlers, and closures queued
// through apply), so Glyph, which reads these fields every frame, never sees a race.
type state struct {
	in Request

	dir       string
	entries   []entry
	cursor    int
	orgSubdir bool
	shallow   bool

	previewGen  int
	previewText string

	phase      phase
	showRun    bool
	busy       bool
	cancel     context.CancelFunc
	results    []clone.Result
	total      int
	finished   int
	pct        int
	status     string
	keys       string
	ran        bool
	dirDisplay string
	options    string
	title      string

	prompting   bool
	prompt      promptKind
	promptLabel string
	field       glyph.InputState
	notice      string
	dirNote     string
	hint        string

	logw io.Writer

	// seams: production wires these to Glyph; tests make them synchronous.
	spawn   func(func())
	apply   func(func())
	refresh func()
	quit    func()

	// set by wire: push/pop the prompt's key router so typed keys reach the field
	enterPrompt func()
	leavePrompt func()
}

func newState(in Request, logw io.Writer, spawn, apply func(func()), refresh, quit func()) *state {
	s := &state{logw: logw, spawn: spawn, apply: apply, refresh: refresh, quit: quit}
	s.reset(in)
	return s
}

// reset (re)initialises the screen for a new Request. Glyph holds pointers into s, so
// fields are reset in place rather than replacing the struct.
func (s *state) reset(in Request) {
	s.in = in
	s.dir = resolveTarget(in.Target)
	s.orgSubdir = in.OrgSubdir
	s.shallow = in.Shallow
	s.title = fmt.Sprintf("Clone %d %s", len(in.Repos), plural(len(in.Repos), "repo", "repos"))
	s.hint = dialogHints
	s.phase, s.showRun, s.busy, s.cancel = phaseDialog, false, false, nil
	s.results, s.total, s.finished, s.pct, s.status, s.keys, s.ran = nil, 0, 0, 0, "", "", false
	s.prompting, s.prompt, s.promptLabel, s.notice = false, 0, "", ""
	s.field = glyph.InputState{}
	s.previewText = ""
	s.reload("")
	s.requestPreview()
}

// resolveTarget makes the starting directory absolute; an empty Target (zero-config)
// starts in the current directory.
func resolveTarget(target string) string {
	p := expandHome(target)
	if p == "" {
		p = "."
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

func relTo(dir, dest string) string {
	if rel, err := filepath.Rel(dir, dest); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return shortenHome(dest)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

func shortenHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}

// reload lists s.dir: ".." (unless at the root), then visible subdirectories. A
// directory that does not exist yet (a fresh configured Target) lists as empty rather
// than an error: clone creates it, and ".." still lets the user climb out. cursorOn, if
// non-empty, is the child name to leave the cursor on (the one just climbed out of).
func (s *state) reload(cursorOn string) {
	s.entries = s.entries[:0]
	if filepath.Dir(s.dir) != s.dir {
		s.entries = append(s.entries, newEntry("", true))
	}
	var names []string
	if des, err := os.ReadDir(s.dir); err == nil {
		for _, de := range des {
			if strings.HasPrefix(de.Name(), ".") {
				continue
			}
			if de.IsDir() {
				names = append(names, de.Name())
			} else if de.Type()&os.ModeSymlink != 0 {
				if fi, err := os.Stat(filepath.Join(s.dir, de.Name())); err == nil && fi.IsDir() {
					names = append(names, de.Name())
				}
			}
		}
	}
	sort.Strings(names)
	for _, n := range names {
		s.entries = append(s.entries, newEntry(n, false))
	}
	s.cursor = 0
	for i, e := range s.entries {
		if cursorOn != "" && e.Name == cursorOn {
			s.cursor = i
		}
	}
	s.dirDisplay = shortenHome(s.dir)
	s.dirNote = ""
	if _, err := os.Stat(s.dir); errors.Is(err, fs.ErrNotExist) {
		s.dirNote = "new folder, created when cloning"
	}
	s.updateOptions()
}

// interactive is true while the dialog accepts browsing keys: not running, not typing
// into the prompt.
func (s *state) interactive() bool {
	return s.phase == phaseDialog && !s.prompting
}

func (s *state) updateOptions() {
	s.options = fmt.Sprintf("org subdirectory: %s · shallow: %s · parallelism: %d", onOff(s.orgSubdir), onOff(s.shallow), s.in.Parallelism)
	if s.dirNote != "" {
		s.options += " · " + s.dirNote
	}
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// open descends into the highlighted directory, or climbs for "..".
func (s *state) open() {
	if !s.interactive() || len(s.entries) == 0 {
		return
	}
	e := s.entries[s.cursor]
	if e.Up {
		s.up()
		return
	}
	s.dir = filepath.Join(s.dir, e.Name)
	s.reload("")
	s.requestPreview()
}

func (s *state) up() {
	if !s.interactive() {
		return
	}
	parent := filepath.Dir(s.dir)
	if parent == s.dir {
		return
	}
	from := filepath.Base(s.dir)
	s.dir = parent
	s.reload(from)
	s.requestPreview()
}

func (s *state) move(delta int) {
	if !s.interactive() || len(s.entries) == 0 {
		return
	}
	s.cursor = min(max(s.cursor+delta, 0), len(s.entries)-1)
}

func (s *state) jump(last bool) {
	if !s.interactive() || len(s.entries) == 0 {
		return
	}
	s.cursor = 0
	if last {
		s.cursor = len(s.entries) - 1
	}
}

func (s *state) toggleOrgSubdir() {
	if !s.interactive() {
		return
	}
	s.orgSubdir = !s.orgSubdir
	s.updateOptions()
	s.requestPreview()
}

// toggleShallow needs no new preview: depth changes how a Repo is cloned, not where or
// whether.
func (s *state) toggleShallow() {
	if !s.interactive() {
		return
	}
	s.shallow = !s.shallow
	s.updateOptions()
}

// requestPreview recomputes the pre-flight preview off the render goroutine. Stale
// answers (the user kept navigating) are dropped by generation.
func (s *state) requestPreview() {
	s.previewGen++
	gen, dir, sub := s.previewGen, s.dir, s.orgSubdir
	s.previewText = "classifying..."
	s.spawn(func() {
		results := s.in.Preview(context.Background(), dir, s.in.Repos, sub)
		s.apply(func() {
			if gen != s.previewGen {
				return
			}
			s.previewText = formatPreview(dir, results)
			s.refresh()
		})
	})
}

// formatPreview lists each Repo's destination relative to dir, since the header
// already shows dir.
func formatPreview(dir string, results []clone.Result) string {
	by := map[clone.Outcome][]clone.Result{}
	for _, r := range results {
		by[r.Outcome] = append(by[r.Outcome], r)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d to clone · %d skipped · %d conflict\n",
		len(by[clone.OutcomeCloned]), len(by[clone.OutcomeSkipped]), len(by[clone.OutcomeConflict]))
	for _, o := range []struct {
		outcome clone.Outcome
		mark    string
	}{{clone.OutcomeCloned, "+"}, {clone.OutcomeSkipped, "="}, {clone.OutcomeConflict, "!"}} {
		for _, r := range by[o.outcome] {
			fmt.Fprintf(&b, "\n%s %s", o.mark, relTo(dir, r.Dest))
		}
	}
	return b.String()
}

// confirm starts a Clone Run for every Repo in the Request.
func (s *state) confirm() {
	if !s.interactive() {
		return
	}
	s.ran = true
	s.results = nil
	repos := make([]clone.Repo, len(s.in.Repos))
	for i, r := range s.in.Repos {
		r.Shallow = s.shallow
		repos[i] = r
	}
	s.startRun(repos)
}

func (s *state) startRun(repos []clone.Repo) {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.phase = phaseRunning
	s.showRun = true
	s.busy = true
	s.total, s.finished, s.pct = len(repos), 0, 0
	s.status = fmt.Sprintf("cloning 0/%d", s.total)
	s.keys = "esc/^c cancel"
	dir, sub := s.dir, s.orgSubdir
	s.spawn(func() {
		fmt.Fprintf(s.logw, "── %s → %s ──\n", s.title, shortenHome(dir))
		results := s.in.Run(ctx, dir, repos, sub, s.in.Parallelism, s.onEvent)
		s.apply(func() { s.finish(results) })
	})
	s.refresh()
}

// onEvent runs on clone goroutines: it writes the log line, then counts on the render
// goroutine.
func (s *state) onEvent(e clone.Event) {
	fmt.Fprintln(s.logw, formatEvent(e))
	if !e.Done {
		return
	}
	s.apply(func() {
		s.finished++
		s.pct = s.finished * 100 / s.total
		s.status = fmt.Sprintf("cloning %d/%d", s.finished, s.total)
		s.refresh()
	})
}

func formatEvent(e clone.Event) string {
	r := e.Result
	name := r.Repo.Name
	switch {
	case !e.Done:
		return fmt.Sprintf("→ %s  cloning...", name)
	case r.Err != nil:
		return fmt.Sprintf("✗ %s  failed: %s", name, strings.Join(strings.Fields(r.Err.Error()), " "))
	case r.Outcome == clone.OutcomeSkipped:
		return fmt.Sprintf("= %s  skipped, already cloned at %s", name, shortenHome(r.Dest))
	case r.Outcome == clone.OutcomeConflict:
		return fmt.Sprintf("! %s  conflict, %s is occupied", name, shortenHome(r.Dest))
	default:
		return fmt.Sprintf("✓ %s  cloned to %s", name, shortenHome(r.Dest))
	}
}

// finish merges a run's results (a retry replaces earlier failures by Repo) and shows
// the summary.
func (s *state) finish(results []clone.Result) {
	s.cancel = nil
	s.phase = phaseDone
	s.busy = false
	s.results = mergeResults(s.results, results)
	var cloned, skipped, conflict, failed int
	for _, r := range s.results {
		switch {
		case r.Err != nil:
			failed++
		case r.Outcome == clone.OutcomeSkipped:
			skipped++
		case r.Outcome == clone.OutcomeConflict:
			conflict++
		default:
			cloned++
		}
	}
	s.pct = 100
	s.status = fmt.Sprintf("done: %d cloned · %d skipped · %d conflict · %d failed", cloned, skipped, conflict, failed)
	s.keys = "esc done"
	if failed > 0 {
		s.keys = "r retry failed · esc done"
	}
	fmt.Fprintln(s.logw, s.status)
	s.refresh()
}

func mergeResults(old, updated []clone.Result) []clone.Result {
	idx := map[string]int{}
	for i, r := range old {
		idx[r.Repo.Org+"/"+r.Repo.Name] = i
	}
	for _, r := range updated {
		if i, ok := idx[r.Repo.Org+"/"+r.Repo.Name]; ok {
			old[i] = r
		} else {
			old = append(old, r)
		}
	}
	return old
}

func (s *state) retry() {
	if s.phase != phaseDone {
		return
	}
	var failed []clone.Repo
	for _, r := range s.results {
		if r.Err != nil {
			failed = append(failed, r.Repo)
		}
	}
	if len(failed) == 0 {
		return
	}
	s.startRun(failed)
}

// back is Esc/^c: it backs out of the dialog, cancels a running clone (the run then
// finishes normally and lands on the summary), or leaves the finished summary.
func (s *state) back() {
	if s.prompting {
		s.closePrompt()
		return
	}
	switch s.phase {
	case phaseRunning:
		if s.cancel != nil {
			s.cancel()
		}
		s.status = "cancelling..."
	default:
		s.quit()
	}
}

func (s *state) output() Response {
	return Response{Ran: s.ran, Target: s.dir, OrgSubdir: s.orgSubdir, Shallow: s.shallow, Results: s.results}
}

func (s *state) openGoto() { s.openPrompt(promptGoto) }
func (s *state) openNew()  { s.openPrompt(promptNew) }

func (s *state) openPrompt(kind promptKind) {
	if !s.interactive() {
		return
	}
	s.prompt, s.prompting, s.notice = kind, true, ""
	s.field.Value = ""
	s.promptLabel = "New folder:"
	if kind == promptGoto {
		s.promptLabel = "Go to:"
		s.field.Value = strings.TrimSuffix(shortenHome(s.dir), "/") + "/"
	}
	s.field.Cursor = len(s.field.Value)
	s.hint = promptHints
	if s.enterPrompt != nil {
		s.enterPrompt()
	}
}

func (s *state) closePrompt() {
	s.prompting = false
	s.hint = dialogHints
	if s.leavePrompt != nil {
		s.leavePrompt()
	}
}

// submitPrompt jumps to the typed folder. A folder that does not exist yet is fine: it
// is only created when the Clone Run writes into it. A bad entry keeps the prompt open
// with the reason shown.
func (s *state) submitPrompt() {
	if !s.prompting {
		return
	}
	path, err := s.resolvePrompt(strings.TrimSpace(s.field.Value))
	if err != nil {
		s.notice = err.Error()
		return
	}
	s.closePrompt()
	s.notice = ""
	s.dir = path
	s.reload("")
	s.requestPreview()
}

func (s *state) resolvePrompt(text string) (string, error) {
	if text == "" {
		return "", errors.New("type a folder")
	}
	var path string
	if s.prompt == promptNew {
		if !filepath.IsLocal(text) {
			return "", errors.New("use a relative name inside the current folder")
		}
		path = filepath.Join(s.dir, text)
	} else {
		path = expandHome(text)
		if !filepath.IsAbs(path) {
			path = filepath.Join(s.dir, path)
		}
		path = filepath.Clean(path)
	}
	fi, err := os.Stat(path)
	switch {
	case err == nil && !fi.IsDir():
		return "", fmt.Errorf("%s is a file", shortenHome(path))
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("cannot use %s: %s", shortenHome(path), errors.Unwrap(err))
	}
	return path, nil
}
