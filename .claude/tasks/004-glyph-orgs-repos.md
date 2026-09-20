# 004 - Glyph Orgs/Repos view: full shell swap

- status: in_progress
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
2. [ ] `internal/browse` pure state
3. [ ] `internal/ui` Glyph browse view + resize tests
4. [ ] Options menu + facets
5. [ ] Leave prompt / Host switch / Help / Fatal overlays
6. [ ] Embed clone screen (mountable), drop tea.Exec + tty hacks
7. [ ] Cutover main.go, delete internal/tui + bubbletea deps, docs (README, DESIGN, ADR-0008)

`internal/tui` stays compiling until step 7.

## Notes / findings

- Step 0 result (headless, glyph v0.8.0): two FilterLists side by side render fine; after
  `SetView`, `app.Router().HandleUnmatched(...)` routes typing to the focused pane's
  `SetQuery`; re-`Handle("<C-n>")` overrides FilterList's default; `Tab` tick+`SelectNext`
  works via `Selected()` pointer; `Stream().WriteAll` + `Refresh()` keep the query and update
  the "n/total" counter; layout reflows at 100/70/50 columns. No fallback to Filter+List needed.
- Open: both panes show the `>` selection marker (Marker is static); focus must be conveyed
  by border colour, check dynamic border colour in step 3.
