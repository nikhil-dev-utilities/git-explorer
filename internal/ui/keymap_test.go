package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// The README's keybinding table must equal keymapTable row for row, so it cannot
// silently drift from what the running app (and F1 help) says.
func TestReadmeKeybindingsTable_MatchesKeymapTable(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	for _, kb := range keymapTable {
		want := fmt.Sprintf("| %s | %s | %s |", kb.mode, kb.key, kb.action)
		if !strings.Contains(string(readme), want) {
			t.Errorf("README.md keybindings table is missing or has drifted from keymapTable:\n  %s", want)
		}
	}
}

// ADR-0006: the filter box is always live, so keys people expect to edit text with must
// never become app verbs. ^a is deliberately not in this set any more (ADR-0008 makes it
// "tick all"; Home still moves to the start of the filter).
func TestKeymapTable_NeverBindsAForbiddenKey(t *testing.T) {
	forbidden := []string{"^h", "^i", "^m", "^[", "^e", "^u", "^w", "^k", "^z", "^d"}
	for _, kb := range keymapTable {
		for _, part := range strings.Split(kb.key, ",") {
			for _, f := range forbidden {
				if strings.EqualFold(strings.TrimSpace(part), f) {
					t.Errorf("keymapTable binds forbidden key %s: %+v", f, kb)
				}
			}
		}
	}
}

// Every key named in the footer hints must exist in the Browse rows.
func TestBrowseKeyHintsAreInTheKeymapTable(t *testing.T) {
	var browse strings.Builder
	for _, kb := range keymapTable {
		if kb.mode == "Browse" {
			browse.WriteString(kb.key + " " + kb.action + " ")
		}
	}
	s := newTest(t, defaultFake(), true)
	for _, f := range []focus{focusOrgs, focusRepos} {
		s.focus = f
		for _, token := range []string{"↑↓", "tab", "^a", "enter", "esc", "^o", "F1", "^c"} {
			if strings.Contains(s.keyHints(), token) &&
				!strings.Contains(strings.ToLower(browse.String()), strings.ToLower(strings.ReplaceAll(token, "↑↓", "↑/↓"))) {
				t.Errorf("focus %v: hint %q is not in the Browse keymap rows", f, token)
			}
		}
	}
}

func TestNewStatePanicsWithoutHosts(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("newState with zero Hosts did not panic")
		}
	}()
	sync := func(f func()) { f() }
	newState(Deps{Forge: defaultFake()}, sync, sync, func() {})
}
