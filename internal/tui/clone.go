package tui

import (
	"io"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
	"github.com/nikhil-dev-utilities/git-explorer/internal/glyphclone"
)

// CloneScreenFunc hands the terminal to the Glyph clone screen and blocks until the
// user leaves it. The Target picker, the pre-flight preview, and the Clone Run itself
// all live inside it (internal/glyphclone), so this package neither previews nor runs
// clones. It is injected so tests never take over a terminal.
type CloneScreenFunc func(glyphclone.Request) (glyphclone.Response, error)

// cloneScreenDoneMsg is delivered when the clone screen returns and bubbletea has
// taken the terminal back.
type cloneScreenDoneMsg struct {
	resp glyphclone.Response
	err  error
}

// execFunc adapts a func to tea.ExecCommand. Glyph reads os.Stdin and writes the tty
// itself, so the stdio setters are deliberately ignored.
type execFunc func() error

func (f execFunc) Run() error        { return f() }
func (execFunc) SetStdin(io.Reader)  {}
func (execFunc) SetStdout(io.Writer) {}
func (execFunc) SetStderr(io.Writer) {}

// enterCloneDialog suspends bubbletea and runs the clone screen for the current
// Selection. Both paths that reach it — Enter on a non-empty Selection, and
// LeavePrompt's "clone now" — go through here.
func (m Model) enterCloneDialog() (Model, tea.Cmd) {
	req := m.cloneRequest()
	screen := m.cloneScreen
	var resp glyphclone.Response
	cmd := tea.Exec(execFunc(func() (err error) {
		resp, err = screen(req)
		return err
	}), func(err error) tea.Msg {
		return cloneScreenDoneMsg{resp: resp, err: err}
	})
	m.mode = ModeBrowse
	return m, cmd
}

func (m Model) cloneRequest() glyphclone.Request {
	return glyphclone.Request{
		Target:      m.cloneTarget,
		OrgSubdir:   m.cloneOrgSubdir,
		Repos:       m.selectedCloneRepos(),
		Parallelism: m.cloneParallelism,
	}
}

// selectedCloneRepos converts the Selection (ticked names within m.repos) into
// clone.Repo values, resolving each CloneURL once via the injected Forge —
// internal/clone never computes one itself (ADR-0001 / PRD 3's own design).
func (m Model) selectedCloneRepos() []clone.Repo {
	var out []clone.Repo
	for _, r := range m.repos {
		if !m.selected[r.Name] {
			continue
		}
		out = append(out, clone.Repo{
			Org:      r.Org,
			Name:     r.Name,
			CloneURL: m.forge.CloneURL(r),
		})
	}
	return out
}

// handleCloneScreenDone remembers the Target and org-subdirectory choice for the
// next Clone Run in this session (see ADR-0007: nothing persists across launches),
// and clears the Selection only if a run actually happened — backing out of the
// screen leaves it intact.
func (m Model) handleCloneScreenDone(msg cloneScreenDoneMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		slog.Error("clone screen failed", "error", msg.err)
		return m, nil
	}
	m.cloneTarget = msg.resp.Target
	m.cloneOrgSubdir = msg.resp.OrgSubdir
	if msg.resp.Ran {
		m.selected = nil
	}
	return m, nil
}
