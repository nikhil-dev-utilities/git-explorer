package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// button is one action a modal dialog offers, rendered as a bracketed
// "[key] Label" widget rather than the bare "[key] label" text hints those
// dialogs used before. primary marks the safe/likely-default action, when a
// dialog has one — visually distinguished (bold, the same focusedBorderColor the
// bordered panes already use for "the thing your attention should land on") so it
// reads at a glance, not just from position in the row.
//
// This is a rendering-only concept: every key named here must already be wired in
// update.go exactly as before. No key-handling logic changes because a button
// exists.
type button struct {
	key     string
	label   string
	primary bool
}

// primaryButtonStyle reuses focusedBorderColor rather than introducing a third
// color — one color already means "pay attention here" via the pane borders; a
// primary button is the same kind of signal.
var primaryButtonStyle = lipgloss.NewStyle().Bold(true).Foreground(focusedBorderColor)

// renderButtons renders a row of button-style widgets, space-separated.
func renderButtons(buttons []button) string {
	parts := make([]string, len(buttons))
	for i, b := range buttons {
		text := fmt.Sprintf("[%s] %s", b.key, b.label)
		if b.primary {
			text = primaryButtonStyle.Render(text)
		}
		parts[i] = text
	}
	return strings.Join(parts, "  ")
}
