package tui

import "testing"

func TestVisibleWindow_UnboundedWhenMaxRowsIsZeroOrLess(t *testing.T) {
	items := []int{0, 1, 2, 3, 4}
	for _, maxRows := range []int{0, -1, -100} {
		window, offset := visibleWindow(items, 2, maxRows)
		if len(window) != len(items) || offset != 0 {
			t.Errorf("visibleWindow(items, 2, %d) = (%v, %d), want (all items, 0)", maxRows, window, offset)
		}
	}
}

func TestVisibleWindow_ShorterThanMaxRowsIsUnaffected(t *testing.T) {
	items := []int{0, 1, 2}
	window, offset := visibleWindow(items, 1, 10)
	if len(window) != 3 || offset != 0 {
		t.Errorf("visibleWindow() = (%v, %d), want (all 3 items, 0) — list already fits", window, offset)
	}
}

func TestVisibleWindow_CursorAlwaysWithinTheReturnedWindow(t *testing.T) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}
	const maxRows = 20

	for _, cursor := range []int{0, 1, 10, 500, 989, 998, 999} {
		window, offset := visibleWindow(items, cursor, maxRows)
		if len(window) != maxRows {
			t.Fatalf("cursor=%d: len(window) = %d, want %d", cursor, len(window), maxRows)
		}
		if cursor < offset || cursor >= offset+len(window) {
			t.Errorf("cursor=%d, offset=%d, len(window)=%d: cursor falls outside the returned window", cursor, offset, len(window))
		}
		// The window must actually be the right slice of the source, at the
		// claimed offset — not just the right length.
		for i, v := range window {
			if v != items[offset+i] {
				t.Fatalf("window[%d] = %d, want items[%d] = %d", i, v, offset+i, items[offset+i])
			}
		}
	}
}

func TestVisibleWindow_NeverStartsBeforeZeroOrEndsPastTheList(t *testing.T) {
	items := make([]int, 50)
	for i := range items {
		items[i] = i
	}
	const maxRows = 10

	// Cursor at the very start: offset must clamp to 0, not go negative.
	_, offset := visibleWindow(items, 0, maxRows)
	if offset != 0 {
		t.Errorf("cursor=0: offset = %d, want 0", offset)
	}

	// Cursor at the very end: window must still be exactly maxRows long, ending
	// exactly at the list's end, not overrunning it.
	window, offset := visibleWindow(items, len(items)-1, maxRows)
	if offset+len(window) != len(items) {
		t.Errorf("cursor=last: offset(%d)+len(window)(%d) = %d, want %d (the list length)", offset, len(window), offset+len(window), len(items))
	}
}

func TestVisibleWindow_CursorOutOfBoundsIsClamped(t *testing.T) {
	items := []int{0, 1, 2, 3, 4}
	// Defensive: a cursor value outside [0, len(items)) must not panic or return
	// a nonsensical window — clamp to the nearest valid index.
	window, offset := visibleWindow(items, 999, 3)
	if len(window) != 3 || offset+len(window) != len(items) {
		t.Errorf("out-of-bounds cursor: window=%v offset=%d, want a valid 3-item window ending at the list's end", window, offset)
	}

	window, offset = visibleWindow(items, -5, 3)
	if len(window) != 3 || offset != 0 {
		t.Errorf("negative cursor: window=%v offset=%d, want a valid 3-item window starting at 0", window, offset)
	}
}
