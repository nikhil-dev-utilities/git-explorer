package tui

import tea "github.com/charmbracelet/bubbletea"

// keyBinding is one row of the keymap. keymapTable is the single source both the
// Help screen's View and the regression test in keymap_test.go read — so the two
// can never drift from each other. It is maintained by hand alongside the actual
// key-handling code in update.go, rather than derived from it automatically (Go
// doesn't offer a practical way to extract "which case labels exist in this switch"
// at runtime) — accuracy against the real switch statements depends on updating both
// together, the same discipline this package's tests already lean on elsewhere.
//
// No entry here uses an alt-key (Meta) alias: DESIGN.md names them as a bonus for
// terminals that send Meta for Option, but no slice of this PRD actually wired one
// up — every binding below is a ctrl key or a plain named key (Tab, Enter, Esc, F1,
// arrows, or a single rune in a modal context where letters aren't filter text).
// That's a real gap against DESIGN.md's original keymap table, left for a future
// issue rather than silently claimed as done here.
type keyBinding struct {
	mode   string
	key    string
	action string
}

var keymapTable = []keyBinding{
	{"Browse", "type", "edit the focused pane's filter"},
	{"Browse", "/", "switch the filter to regex"},
	{"Browse", "↑/↓, ^p/^n", "move the cursor"},
	{"Browse", "Enter", "Orgs: descend · Repos: open clone dialog"},
	{"Browse", "Esc", "Repos: back to Orgs · Orgs: clear filter"},
	{"Browse", "Tab", "tick/untick the focused Repo"},
	{"Browse", "^o", "select all Repos matching the filter"},
	{"Browse", "^t", "Orgs: cycle Affiliation · Repos: cycle archived"},
	{"Browse", "^f", "Repos: cycle fork"},
	{"Browse", "^v", "Repos: cycle visibility"},
	{"Browse", "^s", "cycle sort: name ⇄ last activity"},
	{"Browse", "^y", "switch Host"},
	{"Browse", "^r", "retry a pane-scoped load failure"},
	{"Browse", "F1", "this help screen"},
	{"Browse", "^c", "quit"},

	{"LeavePrompt", "c", "clone now"},
	{"LeavePrompt", "d", "discard the Selection"},
	{"LeavePrompt", "Esc", "stay"},
	{"LeavePrompt", "^c", "quit"},

	{"CloneDialog", "Tab", "toggle the org-subdirectory path"},
	{"CloneDialog", "Enter", "confirm and start the Clone Run"},
	{"CloneDialog", "Esc", "cancel, cloning nothing"},
	{"CloneDialog", "^c", "quit"},

	{"CloneRun", "^c", "cancel the run (while in flight) · quit (once done)"},
	{"CloneRun", "r", "retry failed Repos only"},
	{"CloneRun", "Esc", "done — back to Browse"},

	{"HostSwitch", "↑/↓, ^p/^n", "move the cursor"},
	{"HostSwitch", "Enter", "switch to this Host"},
	{"HostSwitch", "Esc", "cancel"},
	{"HostSwitch", "^c", "quit"},

	{"Fatal", "^y", "switch to a different Host"},
	{"Fatal", "^c", "quit"},
}

// openHelp is only ever reached from ModeBrowse (F1 is bound there and nowhere
// else), so closing Help always returns to Browse specifically, not "whatever mode
// it was opened from" — matching the issue's own acceptance criterion.
func (m Model) openHelp() Model {
	m.mode = ModeHelp
	return m
}

func (m Model) closeHelp() Model {
	m.mode = ModeBrowse
	return m
}

func (m Model) handleHelpKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc, tea.KeyF1:
		return m.closeHelp(), nil
	}
	return m, nil
}
