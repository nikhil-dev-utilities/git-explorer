# 001 - v2 branch: Glyph UI rewrite feasibility study

- status: completed
- last updated: 2026-09-20

## Request

User hates current bubbletea UI (buggy, not user friendly). Wants a `v2` branch and a
complete rewrite of the UI on Glyph (useglyph.sh, github.com/kungfusheep/glyph) — starting
with a feasibility study.

## Plan

1. [x] Create `v2` branch from main.
2. [x] Read Glyph source (scratchpad clone): platforms, testing, widgets, focus, overlays.
3. [x] Spike in scratchpad module: two-pane HBox + CheckList + Filter + overlay, headless
       render into Buffer, key dispatch in tests.
4. [x] Map current internal/tui features/tests to Glyph equivalents; list gaps.
5. [x] Write `docs/glyph-feasibility.md` with verdict + migration plan.
6. [x] Commit on v2 (small commits).

## Notes

- Scope of study: only `internal/tui` (~5.4k LOC incl. tests). `internal/forge`, `internal/clone`,
  `internal/config` stay untouched.
- Glyph: Apache-2.0, pre-1.0, macOS+Linux only, Go 1.25.1, no mouse input found.
- git-explorer goreleaser targets darwin+linux only, so no Windows conflict.

## Outcome

- Study in `docs/glyph-feasibility.md`. Verdict: feasible behind a gate; awaiting user decision (Glyph port vs bubbletea view rewrite).
- Spike code lives in the session scratchpad only (not committed).
