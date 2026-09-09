package tui

// visibleWindow returns the slice of items to render so the cursor stays within the
// visible range, plus offset — the index of the first rendered item, needed so a
// caller can translate a window-relative row back to its real index when deciding
// where to draw the cursor marker.
//
// Without this, a pane with more items than fit in its bordered box's content height
// just rendered every item unconditionally: lipgloss's Height() only pads shorter
// content, it never truncates taller content (confirmed directly, not assumed), so
// the box grew far past its intended size. On a real terminal, whose alt-screen
// buffer has no scrollback, that pushed most of the list off-screen — the cursor
// kept moving correctly in the Model, but almost never inside the terminal's actual
// visible window, which reads as "the keybindings don't do anything." Reported live
// against a private Host with thousands of repos, where this stopped being a
// cosmetic edge case and started being the normal case.
//
// maxRows <= 0 means unbounded: return every item, offset 0. This is the sentinel
// callers pass when there's no real terminal height to constrain against (the
// m.width == 0 / m.height == 0 test-only View() path) — matching how every earlier
// slice of this package rendered before any of this existed.
func visibleWindow[T any](items []T, cursor, maxRows int) (window []T, offset int) {
	if maxRows <= 0 || len(items) <= maxRows {
		return items, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(items) {
		cursor = len(items) - 1
	}

	offset = cursor - maxRows/2
	if offset < 0 {
		offset = 0
	}
	if offset+maxRows > len(items) {
		offset = len(items) - maxRows
	}
	return items[offset : offset+maxRows], offset
}
