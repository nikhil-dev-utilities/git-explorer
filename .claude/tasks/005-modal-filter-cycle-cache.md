# 005 - Modal filter, pane cycling, deselect-all, on-demand Repo loading

- status: completed
- last updated: 2026-09-20

## Request

User wants: Tab/Shift-Tab cycle panes; a way to deselect all; stop reloading Repos each time
focus moves to the Repo pane (1000s of repos); asked if Orgs pane reloads on focus (it does
not: Orgs load at start, host switch, explicit reload; the reload came from `descend()`
always calling `loadRepos()`). User also proposed Space to tick, which needs a modal filter.

Decisions (plan: `/Users/nikhil/.claude/plans/lets-redo-the-orgs-happy-truffle.md`):
- modal filter: `/` to type, bare-key verbs otherwise (Space tick, a toggle-all, x clear, r/F5
  reload, o options, ? help, Tab cycles panes); quit stays ^c only
- session cache per Org + refresh key + "loaded Xm ago" in the Repo title (ADR-0007 condition)
- selection guard only when ticks would be lost (different Org, host switch)

## Status

1. [x] modal filter mode + new base keys
2. [x] pane cycling
3. [x] descend/no refetch/session cache/reload/staleness title
4. [x] selection guard, toggle-all, clear selection, menu row
5. [x] keymap/README/DESIGN/ADR-0009
6. [x] pty check with call-counting fake forge

## Notes
- Done in `internal/ui`: `modeFilter` router, `cyclePane`, `descend` no-refetch + `openOrg` with a
  per-session `repoCache`, `guardSelection`, `toggleAllMatching`, `clearSelection`, `Clear selection`
  menu row, `updateTitles` ("loaded Xm ago"), new keymap/README/DESIGN, ADR-0009 (+ notes on 0005-0008).
- Verified in a real pty with a call-counting fake forge: first Tab fetched once; Tab/Shift-Tab/Space/
  ticks/filter/`a`/`x` never refetched; guard prompt on a different Org; cached revisit stayed at 2 calls;
  `r` and F5 refetched (3, 4); resize; `^c` exit 0.
- Gotcha: the base router must still swallow unmatched keys (`HandleUnmatched` returning false), because
  each FilterList's own text binding is registered there and would type bare letters into a hidden input.
- Answer to "does Orgs load on focus?": no. Only start, host switch, reload. The refetch was
  `descend()` always calling `loadRepos()`.
- Glyph race (not ours): `Log`'s reader goroutine (`syncToLayer` -> `Layer.SetBuffer`) writes without
  synchronising with render (`Layer.blit`); `-race` flags it if a test renders while lines are still being
  appended. Tests now wait 200ms for the reader to go idle before the first render. Worth reporting upstream.
