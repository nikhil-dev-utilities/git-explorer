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

// ADR-0006/0009: while a filter is being typed every printable key is text, so the Filter
// mode must never bind the keys people edit text with. Browse mode has no such limit.
func TestFilterModeNeverBindsATextEditingKey(t *testing.T) {
	forbidden := []string{"^h", "^i", "^m", "^[", "^a", "^e", "^u", "^w", "^k", "^z", "^d"}
	for _, kb := range keymapTable {
		if kb.mode != "Filter" {
			continue
		}
		for _, part := range strings.Split(kb.key, ",") {
			for _, f := range forbidden {
				if strings.EqualFold(strings.TrimSpace(part), f) {
					t.Errorf("Filter mode binds text-editing key %s: %+v", f, kb)
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
			browse.WriteString(strings.ToLower(kb.key) + " ")
		}
	}
	tokens := map[string]string{ // as shown in the hint -> as listed in the table
		"/": "/", "↑↓": "↑/↓", "enter": "enter", "tab": "tab", "space": "space", "a": "a",
		"x": "x", "r": "r", "o": "o", "?": "?", "^c": "^c",
	}
	s := newTest(t, defaultFake(), true)
	for _, f := range []focus{focusOrgs, focusRepos} {
		s.focus = f
		hint := s.keyHints()
		for _, part := range strings.Split(hint, " · ") {
			key := strings.Fields(part)[0]
			want, ok := tokens[key]
			if !ok {
				t.Errorf("focus %v: hint %q uses key %q the test does not know", f, part, key)
				continue
			}
			if !strings.Contains(browse.String(), want) {
				t.Errorf("focus %v: hint key %q is not in the Browse keymap rows", f, key)
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
