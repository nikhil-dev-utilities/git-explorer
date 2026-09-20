package ui

// keyBinding is one row of the keymap. keymapTable is the single source the Help screen
// renders and the README table is checked against, so they cannot drift.
type keyBinding struct {
	mode   string
	key    string
	action string
}

var keymapTable = []keyBinding{
	{"Browse", "type", "filter the focused pane (fzf: 'exact ^start end$ !not)"},
	{"Browse", "↑/↓, ^p/^n", "move the cursor"},
	{"Browse", "PgUp/PgDn", "move a page"},
	{"Browse", "Enter, →", "Orgs: open the Org · Repos: clone the ticked Repos"},
	{"Browse", "Esc, ←", "Repos: back to Orgs · Orgs: clear the filter"},
	{"Browse", "Tab, Shift-Tab", "tick the Repo and move down / up"},
	{"Browse", "^a", "tick every Repo matching the filter"},
	{"Browse", "^o", "options: host, facets, sort, pane width, reload"},
	{"Browse", "F1", "this help"},
	{"Browse", "^c", "quit"},

	{"LeavePrompt", "c", "clone now"},
	{"LeavePrompt", "d", "discard the Selection"},
	{"LeavePrompt", "Esc", "stay"},
	{"LeavePrompt", "^c", "quit"},

	{"Options", "↑/↓", "move"},
	{"Options", "Enter, Space", "change the value"},
	{"Options", "Esc, ^o", "close"},

	{"HostSwitch", "↑/↓, ^p/^n", "move the cursor"},
	{"HostSwitch", "Enter", "switch to this Host"},
	{"HostSwitch", "Esc", "cancel"},
	{"HostSwitch", "^c", "quit"},

	{"Fatal", "^y", "switch to a different Host"},
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
