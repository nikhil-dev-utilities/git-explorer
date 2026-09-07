package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestReadmeKeybindingsTable_MatchesKeymapTable guards the one thing a hand-copied
// README table can silently drift on: every row must still say exactly what
// keymapTable says. It renders each row in the same "| Mode | Key | Alt | Action |"
// format the README's own keybindings table uses and requires each one to appear
// verbatim — so adding, removing, or editing a binding in keymapTable without also
// updating README.md fails this test rather than shipping stale docs.
func TestReadmeKeybindingsTable_MatchesKeymapTable(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	content := string(readme)

	for _, kb := range keymapTable {
		want := fmt.Sprintf("| %s | %s | %s | %s |", kb.mode, kb.key, kb.altKey, kb.action)
		if !strings.Contains(content, want) {
			t.Errorf("README.md's keybindings table is missing or has drifted from this keymapTable row:\n  %s", want)
		}
	}
}
