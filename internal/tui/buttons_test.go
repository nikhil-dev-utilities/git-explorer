package tui

import (
	"strings"
	"testing"
)

// primaryButtonStyle's actual rendered ANSI output is stripped outside a real
// terminal (same lipgloss behavior noted in pane_border_test.go), so its Bold/Color
// are asserted against the Style object's own getters rather than scraped text.
func TestPrimaryButtonStyle_IsBoldAndUsesFocusedColor(t *testing.T) {
	if !primaryButtonStyle.GetBold() {
		t.Error("primaryButtonStyle.GetBold() = false, want true")
	}
	if got := primaryButtonStyle.GetForeground(); got != focusedBorderColor {
		t.Errorf("primaryButtonStyle.GetForeground() = %v, want focusedBorderColor", got)
	}
}

func TestRenderButtons_IncludesEveryKeyAndLabel(t *testing.T) {
	buttons := []button{
		{key: "c", label: "clone now"},
		{key: "d", label: "discard"},
		{key: "esc", label: "stay", primary: true},
	}
	got := renderButtons(buttons)
	for _, b := range buttons {
		if !strings.Contains(got, b.key) {
			t.Errorf("renderButtons() = %q, want it to contain key %q", got, b.key)
		}
		if !strings.Contains(got, b.label) {
			t.Errorf("renderButtons() = %q, want it to contain label %q", got, b.label)
		}
	}
}

// The each-dialog button-list functions are tested for content (right keys, right
// labels, right one marked primary) — matched against update.go's real key handling
// by the existing behavioral tests (TestLeavePrompt_*, TestCloneDialog_*,
// TestHostSwitch_*, TestFatal_*), which exercise the actual keys, not this list.

func TestLeavePromptButtons(t *testing.T) {
	assertExactlyOnePrimary(t, leavePromptButtons(), "esc")
	assertHasKeys(t, leavePromptButtons(), "c", "d", "esc")
}

func TestCloneDialogButtons(t *testing.T) {
	assertExactlyOnePrimary(t, cloneDialogButtons(), "esc")
	assertHasKeys(t, cloneDialogButtons(), "enter", "esc")
}

func TestHostSwitchButtons(t *testing.T) {
	assertExactlyOnePrimary(t, hostSwitchButtons(), "esc")
	assertHasKeys(t, hostSwitchButtons(), "enter", "esc", "^c")
}

func TestFatalButtons_HasNoPrimary(t *testing.T) {
	for _, b := range fatalButtons() {
		if b.primary {
			t.Errorf("fatalButtons() has %q marked primary, want none — neither escape is a safer default than the other", b.key)
		}
	}
	assertHasKeys(t, fatalButtons(), "^y", "^c")
}

func assertHasKeys(t *testing.T, buttons []button, keys ...string) {
	t.Helper()
	got := map[string]bool{}
	for _, b := range buttons {
		got[b.key] = true
	}
	for _, k := range keys {
		if !got[k] {
			t.Errorf("buttons = %+v, want key %q present", buttons, k)
		}
	}
}

func assertExactlyOnePrimary(t *testing.T, buttons []button, wantKey string) {
	t.Helper()
	var primaryKeys []string
	for _, b := range buttons {
		if b.primary {
			primaryKeys = append(primaryKeys, b.key)
		}
	}
	if len(primaryKeys) != 1 || primaryKeys[0] != wantKey {
		t.Errorf("primary buttons = %v, want exactly [%q]", primaryKeys, wantKey)
	}
}
