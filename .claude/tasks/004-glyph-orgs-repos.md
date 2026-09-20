# 004 - Glyph Orgs/Repos view: full shell swap

- status: completed
- last updated: 2026-09-20

## Request

Redo the Orgs and Repos view on Glyph using the FilterList (fuzzy finder example) for both
panes. Asked whether Glyph redraws panes on window resize (yes: SIGWINCH -> per-frame layout).
User approved (plan: `/Users/nikhil/.claude/plans/lets-redo-the-orgs-happy-truffle.md`):
- full shell swap: Glyph is the only UI, delete bubbletea/lipgloss/teatest + tty workarounds
- fzf query syntax replaces substring + /regex
- new keymap (fewer chords): type=filter, arrows, Enter/->, Esc/<-, Tab tick+advance,
  ^a tick all matches, ^o options menu (host, archived, forks, visibility, sort, width,
  reload), F1 help, ^c quit

## Plan / status

1. [x] Step 0 spike (scratchpad): two FilterLists, routing override, SetQuery/Refresh, Stream
2. [x] pure state (merged into `internal/ui/state.go`, no separate `browse` pkg)
3. [x] `internal/ui` Glyph browse view + resize tests
4. [x] Options menu + facets
5. [x] Leave prompt / Host switch / Help / Fatal overlays
6. [x] Embed clone screen (mountable), drop tea.Exec + tty hacks
7. [x] Cutover main.go, delete internal/tui + bubbletea deps, docs (README, DESIGN, ADR-0008)

`internal/tui` stays compiling until step 7.

## Notes / findings

- Step 0 result (headless, glyph v0.8.0): two FilterLists side by side render fine; after
  `SetView`, `app.Router().HandleUnmatched(...)` routes typing to the focused pane's
  `SetQuery`; re-`Handle("<C-n>")` overrides FilterList's default; `Tab` tick+`SelectNext`
  works via `Selected()` pointer; `Stream().WriteAll` + `Refresh()` keep the query and update
  the "n/total" counter; layout reflows at 100/70/50 columns. No fallback to Filter+List needed.
- Open: both panes show the `>` selection marker (Marker is static); focus must be conveyed
  by border colour, check dynamic border colour in step 3.
- Glyph findings while building the browse view (v0.8.0):
  - FilterLists must NOT sit inside an `If` branch: conditional branches wire bindings through
    child scopes that sit ahead of our post-`SetView` routing override (typing went to the
    wrong pane, and the Repo pane sized to content). Keep panes at the root; full-screen states
    are `Overlay` cards over them.
  - `selectionList.len` is only set during render, so `SelectNext/Prev` are no-ops until a
    frame has been drawn. Headless tests must `Execute` a frame before cursor-moving keys.
  - Panes need an explicit `Height(&paneH)` (rows minus footer) on the HBox and both panes,
    otherwise a short terminal lets content overflow the footer and hides bottom borders.
  - Rows: avoid `HBox.Gap` with `If` children (an empty branch still gets a gap, clipping the
    last column by one); use explicit `SpaceW(1)`.
  - `VBox.Title` is static: titles that change (repo pane) are a `Text(&title)` inside the box.

## Outcome

- Done and committed on v2: `internal/ui` (state, view, keymap, options/host/help/leave/fatal
  cards), embedded clone screen (`glyphclone.Embed`), cutover in `cmd/git-explorer`,
  `internal/tui` and bubbletea/lipgloss/teatest removed, tty/TIOCSTI workarounds removed,
  README/DESIGN/ADR-0008 updated.
- Verified in a real pty (pyte) with a fake forge: streaming Orgs with spinner, per-pane
  filter, focus, ticks, live resize (reflow, age then badges shed, too-narrow card), options
  overlay, clone screen + run, back to browse, `^c` exits status 0.
- Not verified: Linux; real `gh` end to end (needs the user's credentials).
- Follow-ups worth filing: Tab completion in the clone path prompt; unfocused pane still
  shows a `>` marker (focus is border colour only); non-ASCII filter/path editing untested.
