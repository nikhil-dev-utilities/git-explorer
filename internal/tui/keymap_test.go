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

// TestKeymapTable_EveryActionHasACtrlOrNamedKey guards ADR-0006's other half: an
// alt-key (Meta) alias must never be the *only* path to an action. Every row's
// primary key column (kb.key) must be a real ctrl/named key regardless of whether
// that row also carries an altKey — an alt alias lives in its own column precisely
// so it can never be entered here as if it were the primary binding.
func TestKeymapTable_EveryActionHasACtrlOrNamedKey(t *testing.T) {
	for _, kb := range keymapTable {
		if strings.Contains(strings.ToLower(kb.key), "alt") {
			t.Errorf("mode %s binds %q via an alt-only key in the primary key column — alt aliases belong in altKey, not key", kb.mode, kb.key)
		}
		if kb.altKey != "" && kb.key == "" {
			t.Errorf("mode %s has altKey %q but no primary ctrl/named key — an alt alias must never be the only path to an action", kb.mode, kb.altKey)
		}
	}
}

// TestKeymapTable_AltKeyAliasesMatchDesign guards the specific alt-key aliases
// DESIGN.md commits to (its "mnemonic alt bindings" table) — a future edit to
// keymapTable can't silently drop or rename one without this failing.
func TestKeymapTable_AltKeyAliasesMatchDesign(t *testing.T) {
	want := map[string]string{ // "mode|key" -> altKey
		"Browse|Enter": "alt-c",
		"Browse|^o":    "alt-a",
		"Browse|^t":    "alt-x",
		"Browse|^f":    "alt-f",
		"Browse|^v":    "alt-v",
		"Browse|^s":    "alt-s",
		"Browse|^g":    "alt-w",
		"Browse|^y":    "alt-h",
		"Browse|^r":    "alt-r",
		"Fatal|^y":     "alt-h",
	}

	got := map[string]string{}
	for _, kb := range keymapTable {
		if kb.altKey != "" {
			got[kb.mode+"|"+kb.key] = kb.altKey
		}
	}

	for k, wantAlt := range want {
		if gotAlt, ok := got[k]; !ok {
			t.Errorf("%s: no altKey entry found, want %q", k, wantAlt)
		} else if gotAlt != wantAlt {
			t.Errorf("%s: altKey = %q, want %q", k, gotAlt, wantAlt)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s: has an altKey not in DESIGN.md's table (or this test's expectations are stale)", k)
		}
	}
}
