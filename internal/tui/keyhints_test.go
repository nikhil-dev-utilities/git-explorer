package tui

import (
	"strings"
	"testing"
)

// TestBrowseKeyHints_EveryKeyIsInKeymapTable guards currentBrowseKeyHints against
// drifting from the real bindings: every key it lists must actually be a Browse-mode
// row in keymapTable, in both possible focus states (Enter/Esc's labels change with
// focus, but the key set itself doesn't).
func TestBrowseKeyHints_EveryKeyIsInKeymapTable(t *testing.T) {
	browseKeys := map[string]bool{}
	for _, kb := range keymapTable {
		if kb.mode == "Browse" {
			browseKeys[kb.key] = true
		}
	}

	for _, focus := range []Focus{FocusOrgs, FocusRepos} {
		m := Model{focus: focus}
		for _, h := range m.currentBrowseKeyHints() {
			if !browseKeys[h.key] {
				t.Errorf("currentBrowseKeyHints() (focus=%v) includes key %q, which is not a Browse row in keymapTable", focus, h.key)
			}
		}
	}
}

func TestRenderKeyHintGrid_WrapsToMultipleColumns(t *testing.T) {
	hints := []keyHint{{"a", "one"}, {"b", "two"}, {"c", "three"}, {"d", "four"}}

	// width 80 / keyHintCellWidth(18) = 4 columns — all 4 hints fit on one row.
	grid := renderKeyHintGrid(hints, 80)
	if got := strings.Count(grid, "\n"); got != 1 {
		t.Errorf("row count = %d, want 1 (4 hints, 4 columns fit at width 80)", got)
	}

	// width 45 / keyHintCellWidth(20) = 2 columns — 4 hints wrap to 2 rows.
	grid = renderKeyHintGrid(hints, 45)
	if got := strings.Count(grid, "\n"); got != 2 {
		t.Errorf("row count = %d, want 2 (4 hints, 2 columns fit at width 45)", got)
	}

	for _, h := range hints {
		if !strings.Contains(grid, h.key) || !strings.Contains(grid, h.label) {
			t.Errorf("grid = %q, want it to contain both %q and %q", grid, h.key, h.label)
		}
	}
}

func TestRenderKeyHintGrid_NeverFewerThanOneColumn(t *testing.T) {
	hints := []keyHint{{"a", "one"}, {"b", "two"}}

	// A width narrower than one cell must still render — one column, one hint per row.
	grid := renderKeyHintGrid(hints, 5)
	if got := strings.Count(grid, "\n"); got != 2 {
		t.Errorf("row count = %d, want 2 (one column when width can't fit even one full cell)", got)
	}
}

func TestKeyHintGridRows_MatchesRenderKeyHintGrid(t *testing.T) {
	hints := []keyHint{{"a", "one"}, {"b", "two"}, {"c", "three"}, {"d", "four"}, {"e", "five"}}

	for _, width := range []int{5, 20, 36, 54, 80, 200} {
		wantRows := strings.Count(renderKeyHintGrid(hints, width), "\n")
		if got := keyHintGridRows(hints, width); got != wantRows {
			t.Errorf("keyHintGridRows(width=%d) = %d, want %d (must match renderKeyHintGrid's actual row count)", width, got, wantRows)
		}
	}
}
