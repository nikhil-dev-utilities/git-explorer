package tui

import (
	"strings"
	"testing"
)

// TestKeymapTable_NeverBindsAForbiddenKey is the regression guard ADR-0006 exists to
// be protected by: none of these keys may ever be bound to an app verb, in any mode.
// ^h is backspace, ^i is Tab, ^m/^[ are Enter/Esc, ^a/^e/^u/^w/^k are readline
// home/end/kill (expected to work inside the always-focused filter box), and ^z/^d
// belong to the terminal. ^c is deliberately excluded from this forbidden set: it is
// bound throughout this package, but only ever to quit (or cancel-then-quit in
// CloneRun) — the same meaning it already has in virtually every terminal program,
// not a new app-specific verb riding on a key someone would expect to interrupt the
// process.
//
// This checks keymapTable, the same single source viewHelp renders from — not the
// key-handling switch statements in update.go directly, which Go has no practical
// way to introspect at runtime. Keeping the table accurate against those switches is
// a discipline, not something this test can fully guarantee on its own; a handful of
// other tests in this package (e.g. TestBrowse_BackspaceEditsFilter, which proves ^h
// edits text rather than triggering a verb) corroborate specific entries directly
// against real key delivery through a running Program.
func TestKeymapTable_NeverBindsAForbiddenKey(t *testing.T) {
	forbidden := []string{"^h", "^i", "^m", "^[", "^a", "^e", "^u", "^w", "^k", "^z", "^d"}

	for _, kb := range keymapTable {
		keyLower := strings.ToLower(kb.key)
		for _, f := range forbidden {
			// Match whole key tokens (keys can be comma-separated, e.g. "↑/↓, ^p/^n")
			// rather than a bare substring, so "^enter" style false positives can't
			// sneak past a check that only looked for "^e" as a substring.
			for _, tok := range strings.FieldsFunc(keyLower, func(r rune) bool {
				return r == ',' || r == '/' || r == ' '
			}) {
				if tok == f {
					t.Errorf("mode %s binds forbidden key %q (action: %q) — ADR-0006 reserves this for the terminal/readline/text-editing, never an app verb",
						kb.mode, kb.key, kb.action)
				}
			}
		}
	}
}

// TestKeymapTable_CtrlCIsOnlyEverQuitOrCancel guards the one exception in the test
// above: ^c must never be reused for an unrelated verb in any mode.
func TestKeymapTable_CtrlCIsOnlyEverQuitOrCancel(t *testing.T) {
	for _, kb := range keymapTable {
		if kb.key != "^c" {
			continue
		}
		if !strings.Contains(strings.ToLower(kb.action), "quit") && !strings.Contains(strings.ToLower(kb.action), "cancel") {
			t.Errorf("mode %s binds ^c to %q, want it to only ever mean quit or cancel", kb.mode, kb.action)
		}
	}
}

// TestKeymapTable_EveryActionHasACtrlOrNamedKey guards ADR-0006's other half: any
// alt-key (Meta) alias must never be the *only* path to an action. Since no slice of
// this PRD actually wired up alt-key aliases (see help.go's doc comment — a real gap
// against DESIGN.md, left for a future issue), this is currently satisfied
// vacuously: there are no "Alt+" entries in the table at all, so nothing can depend
// on one exclusively. This test exists so it starts failing the moment someone adds
// an alt-only entry without also adding the ctrl/named-key counterpart.
func TestKeymapTable_EveryActionHasACtrlOrNamedKey(t *testing.T) {
	for _, kb := range keymapTable {
		if strings.Contains(strings.ToLower(kb.key), "alt") {
			t.Errorf("mode %s binds %q via an alt-only key with no corresponding ctrl/named-key entry found by this simple table scan", kb.mode, kb.key)
		}
	}
}
