package tui

import (
	"fmt"
	"strings"
)

// keyHint is one cell of the nano/mc-style key-hint grid: a short label, distinct
// from keymapTable's longer descriptive prose (which is written for the F1 Help
// screen, a different rendering surface with different space constraints).
type keyHint struct {
	key   string
	label string
}

// keyHintCellWidth is how wide one "key  label" cell is padded to, so cells line up
// into a grid regardless of how many columns fit. 20 is the smallest width that
// fits every current label without truncating (the longest, "Esc   clear filter",
// is 18 characters) while leaving at least one column of gap before the next cell.
const keyHintCellWidth = 20

// currentBrowseKeyHints is Browse mode's key-hint grid content. Enter and Esc's
// labels depend on focus, matching what they actually do right now — same
// distinction keymapTable's own prose already draws ("Orgs: descend · Repos: open
// clone dialog"). TestBrowseKeyHints_EveryKeyIsInKeymapTable guards every key here
// against actually existing in keymapTable's Browse rows, so this list can't
// silently drift from the real bindings.
func (m Model) currentBrowseKeyHints() []keyHint {
	enterLabel, escLabel := "descend", "clear filter"
	if m.focus == FocusRepos {
		enterLabel, escLabel = "clone", "back"
	}
	return []keyHint{
		{"Tab", "tick"},
		{"^o", "select all"},
		{"Enter", enterLabel},
		{"Esc", escLabel},
		{"^t", "cycle facet"},
		{"^f", "cycle fork"},
		{"^v", "cycle vis"},
		{"^s", "sort"},
		{"^y", "switch host"},
		{"^r", "retry"},
		{"F1", "help"},
		{"^c", "quit"},
	}
}

// renderKeyHintGrid packs hints into as many columns as width affords — nano/mc
// style, wrapping to further rows rather than truncating the list or overflowing
// the terminal. Always at least one column. Every row, including the last, ends in
// a newline, so a caller can safely count rows via keyHintGridRows before this is
// ever rendered (needed to size the panes above it).
func renderKeyHintGrid(hints []keyHint, width int) string {
	cols := hintGridCols(width)

	var b strings.Builder
	for i, h := range hints {
		cell := fmt.Sprintf("%-5s %s", h.key, h.label)
		if len(cell) > keyHintCellWidth-1 {
			cell = cell[:keyHintCellWidth-1]
		}
		fmt.Fprintf(&b, "%-*s", keyHintCellWidth, cell)
		if (i+1)%cols == 0 || i == len(hints)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// keyHintGridRows returns how many rows renderKeyHintGrid(hints, width) will
// occupy, without building the string.
func keyHintGridRows(hints []keyHint, width int) int {
	cols := hintGridCols(width)
	rows := len(hints) / cols
	if len(hints)%cols != 0 {
		rows++
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

func hintGridCols(width int) int {
	cols := width / keyHintCellWidth
	if cols < 1 {
		cols = 1
	}
	return cols
}
