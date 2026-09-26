package ui

// keyBinding is one row of the keymap. keymapTable is the single source the Help screen
// renders and the README table is checked against, so they cannot drift.
type keyBinding struct {
	mode   string
	key    string
	action string
}

var keymapTable = []keyBinding{
	{"Browse", "/", "filter the focused pane (fzf: 'exact ^start end$ !not, space = AND)"},
	{"Browse", "↑/↓, j/k, ^p/^n", "move the cursor"},
	{"Browse", "PgUp/PgDn", "move a page"},
	{"Browse", "Tab, Shift-Tab", "switch between the Org and Repo panes (never reloads)"},
	{"Browse", "Enter, →, l", "Orgs: open the Org (the Org already shown is not refetched)"},
	{"Browse", "Enter, c", "Repos: clone the ticked Repos"},
	{"Browse", "Esc", "clear the pane's filter; with none, Repos back to Orgs"},
	{"Browse", "←, h", "Repos back to Orgs (ticks and filter are kept)"},
	{"Browse", "Space", "tick the Repo and move down"},
	{"Browse", "a", "tick every Repo matching the filter; again to untick them"},
	{"Browse", "x", "clear the whole Selection, including ticks the filter hides"},
	{"Browse", "r, F5", "reload the focused pane from the Forge"},
	{"Browse", "o", "options: host, facets, sort, pane width, clear selection, reload"},
	{"Browse", "?, F1", "this help"},
	{"Browse", "^c", "quit"},

	{"Filter", "type", "edit the focused pane's filter (every key is text)"},
	{"Filter", "↑/↓, ^p/^n, PgUp/PgDn", "move the list while typing"},
	{"Filter", "Enter", "accept: back to the list, filter kept"},
	{"Filter", "Esc", "clear the filter and go back to the list"},
	{"Filter", "^c", "quit"},

	{"LeavePrompt", "c", "clone the ticked Repos now"},
	{"LeavePrompt", "d", "discard the Selection and continue"},
	{"LeavePrompt", "Esc", "stay"},
	{"LeavePrompt", "^c", "quit"},

	{"CloneDialog", "↑/↓, j/k", "move in the folder browser"},
	{"CloneDialog", "Enter", "open the highlighted folder"},
	{"CloneDialog", "←, h, Backspace", "go up a folder"},
	{"CloneDialog", "/", "type a path to jump to (~, absolute or relative); Enter go · Esc cancel"},
	{"CloneDialog", "n", "new folder in this one, made when cloning; Enter create · Esc cancel"},
	{"CloneDialog", "Tab", "toggle the org-subdirectory path"},
	{"CloneDialog", "s", "toggle shallow clones (--depth 1)"},
	{"CloneDialog", "c", "start the Clone Run"},
	{"CloneDialog", "Esc, ^c", "cancel, cloning nothing"},

	{"CloneRun", "Esc, ^c", "cancel the run (while in flight)"},
	{"CloneRun", "r", "retry failed Repos only"},
	{"CloneRun", "Esc", "done: back to Browse"},

	{"Options", "↑/↓", "move"},
	{"Options", "Enter, Space", "change the value"},
	{"Options", "Esc, o", "close"},

	{"HostSwitch", "↑/↓, ^p/^n", "move the cursor"},
	{"HostSwitch", "Enter", "switch to this Host"},
	{"HostSwitch", "Esc", "cancel"},
	{"HostSwitch", "^c", "quit"},

	{"Fatal", "y", "switch to a different Host"},
	{"Fatal", "^c", "quit"},
}

// helpRow is one line of the Help overlay; a section heading has the name in Key only.
type helpRow struct {
	Key  string
	Text string
}

func helpRows() []helpRow {
	var rows []helpRow
	section := ""
	for _, kb := range keymapTable {
		if kb.mode != section {
			section = kb.mode
			if len(rows) > 0 {
				rows = append(rows, helpRow{})
			}
			rows = append(rows, helpRow{Key: section})
		}
		rows = append(rows, helpRow{Key: kb.key, Text: kb.action})
	}
	return rows
}
