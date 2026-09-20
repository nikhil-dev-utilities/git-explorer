# v2: rewriting the TUI on Glyph — feasibility study

Status: this study led to the port; see ADR-0008 and `.claude/tasks/004-glyph-orgs-repos.md`. The
port is complete on branch `v2` (bubbletea removed). Text below is the original study.

## Verdict

**Feasible; recommend proceeding behind one gate.** The rewrite touches only `internal/tui`
(about 2.3k non-test lines, 3.1k test lines). `forge`, `clone`, `config` and the pure
logic inside `tui` (filtering, sorting, selection, Outcome preview) carry over unchanged.
Glyph covers every widget the current UI hand-builds. The three real costs are: pre-1.0
single-maintainer dependency, a Go toolchain bump, and a weaker headless-testing story
than `teatest`.

One caution before starting: a framework swap does not by itself make the UI "less buggy
and more user friendly". Part of the current pain is manual layout arithmetic in
`view.go` (569 lines of height budgets, footer-row counting and border math) — Glyph's flex
layout removes that class of bug. The rest is UX design, which needs its own pass
(see "Do first" below).

## What Glyph is

- `github.com/kungfusheep/glyph`, Apache-2.0, created 2026-01, last push 2026-08-28,
  about 150 stars, 2 open issues, one primary author. README says "pre-1.0".
- Declarative and retained: you build a tree of typed values (`VBox`, `HBox`, `List`,
  `CheckList`, `Input`, `Overlay`, ...) that hold pointers to your state; Glyph re-reads the
  pointers each frame and diff-flushes to the terminal. There is no `Update`/`Msg` loop.
- Keys go through its `riffkey` router (vim-style patterns, `<C-x>`, `<A-x>`, `<Tab>`).
  Goroutines mutate state via `app.Apply(func())`.

## Spike results (throwaway module, Go 1.26, glyph @ a67d7a5)

| Question | Result |
|---|---|
| Two bordered panes, org list + repo checklist with right-aligned age | Works. `HBox.Grow(1)(VBox.WidthPct(.3).Border(..), VBox.Grow(1).Border(..))` |
| Checkbox multi-select | `CheckList(&repos).Checked(func(r *repo) *bool)` + `BindToggle(" ")` works; space toggled the row |
| Modal dialog | `If(&flag).Then(Overlay.Centered()(...))` renders centred over the base view |
| Live filter input while list is visible | `Input(&q)` receives typed runes; list is independent |
| Headless render for tests | `app.Template().Execute(NewBuffer(w,h), w, h)` then `buf.StringTrimmed()` — no TTY needed |
| Key injection for tests | `app.Input().Dispatch(riffkey.Key{Rune: ' '})` works headlessly |
| 5,000-row org list | about 94 µs per render at 100x40. Windowing is built in — this replaces `scroll.go` |
| Goroutine state updates in tests | **Gap.** `Apply` closures drain only inside the unexported render loop, so a test cannot flush them from outside the package |

## Feature mapping (current bubbletea code to Glyph)

| Current | Glyph | Effort |
|---|---|---|
| `Model` / `Update` / `View`, `Mode` enum | Named views (`app.View`, `PushView`/`PopView`) or one view plus `If/Switch` on a mode field | Medium: state machine stays, plumbing changes |
| Two panes, manual border colors and height math (`view.go`, `layout.go`) | `HBox`/`VBox` flex layout, `Border`, focus styling | Small; deletes most of `layout.go` |
| Org/Repo list windowing (`scroll.go`) | Built into `List`/`CheckList` | Deleted |
| Selection with checkmarks (`selection.go`) | `CheckList` + `Checked` accessor | Small |
| Always-focused filter box, ADR-0006 | `Input` plus own routing so nav keys reach the list | Medium — see risk 3 |
| Filter semantics: substring, `/regex`, affiliation, archived/fork/visibility facets | Keep `filter.go` as-is. Glyph's `Filter`/`FilterList` are fzf-fuzzy, which would change behaviour and drop regex | Small; reuse |
| Clone dialog, leave prompt, host switch, fatal, help | `Overlay` + `Text`/`Button`-style rows; `Form` for target path and toggle | Medium |
| Clone Run progress and per-repo Outcomes | `Progress`, `Spinner`, `Log` | Small |
| Key hints footer (`keyhints.go`, `help.go`) | Glyph has `keyhelp.go` and `ActiveBindings()` | Small |
| Async fetch to Msgs | Goroutine plus `app.Apply` | Small |
| Resizable Org pane, narrow-terminal degradation | `WidthPct`/`Grow`, conditional rendering off `OnResize` | Small |
| Mouse | Neither app has it. Glyph has no mouse input either | None |

## Risks

1. **Pre-1.0, single maintainer.** API can move under us. Mitigation: pin an exact
   version; keep all Glyph imports inside `internal/tui` so the swap is contained.
2. **Toolchain bump.** Glyph requires Go 1.25.1; this repo and `ci.yml` pin 1.24.2. Needs
   `go.mod`, CI and goreleaser updated. Low risk, but it is a repo-wide change.
3. **Focus and key routing is the hardest part.** ADR-0006 wants the filter always focused
   while arrows/enter act on the list. In the spike, typing worked, but plain `Input` does
   not forward navigation. Expect to write explicit bindings. Needs a second spike inside
   the real screens before we commit to the full port.
4. **Tests.** 137 test references are coupled to `tea.KeyMsg`/`teatest`. They must be
   rewritten against `Template.Execute` + `Input.Dispatch`. Rendering and key tests work
   (spike-verified); async flows need either a small in-package `Apply` drain upstreamed to
   Glyph, or a state layer we test without the view. Recommend the latter: keep a pure
   `State` struct with `Reduce(event)`, tested directly, with Glyph as a thin view.
5. **Platforms.** Glyph is macOS and Linux only ("Windows on the roadmap"). goreleaser
   already targets darwin and linux only, so no regression, but this becomes a hard
   ceiling if Windows is ever wanted.
6. **Docs and community.** Sparse; expect to read source (as this study did).
7. **fzf dependency.** Glyph pulls in `junegunn/fzf` (matching only). Binary size grows;
   measure in the first slice.

## Alternative worth naming

Keep bubbletea and rewrite only `view.go` with `bubbles` components and a proper layout
pass. Lower risk, mature ecosystem, Windows works, `teatest` keeps working. Glyph wins on
layout ergonomics and built-in widgets; bubbletea wins on stability and testing. If the
complaint is mostly UX rather than framework limits, this option gets most of the benefit.

## Do first, regardless of framework

The two complaints ("buggy", "not user friendly") are not yet itemised. Before porting:

- List the concrete bugs (repro steps) and friction points. File as GitHub issues.
- Decide the target UX (screens, key model, discoverability). The current design has
  `^t ^f ^v ^s ^g ^y ^o` chords plus `alt-` variants; that may itself be the friendliness
  problem and it is independent of the toolchain.

## Proposed plan if approved

1. Extract a framework-free `State` + `Reduce` from `Model` (pure, fully tested). Doable on
   `main` and useful under either framework.
2. `v2`: bump Go, add pinned Glyph, port Browse (two panes, filter, checklist) behind the
   existing `main.go` wiring. Second spike: focus and key routing on real data.
3. Port overlays: clone dialog, leave prompt, host switch, fatal, help.
4. Port clone run and failures.
5. Rewrite tests on the headless harness; delete `bubbletea`/`lipgloss`/`teatest`.
6. Gate: only merge if all current acceptance behaviour (DESIGN.md) is covered.

## Decision needed

Go (proceed with step 1 and 2), or take the bubbletea view-rewrite alternative?
