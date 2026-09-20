# 002 - Glyph port, slice 1: clone view

- status: in_progress
- last updated: 2026-09-20

## Request

User chose the Glyph port (see 001). Start with the clone view, using Glyph components:
- file browser: shows the clone target path and lets the user change it
- deploy log: shows clone output per selected repo

## Design (decided, no user questions needed)

- Glyph and bubbletea cannot share a terminal, so the Glyph clone screen runs as a
  bubbletea `tea.Exec` sub-program: bubbletea releases the terminal, Glyph runs the whole
  clone dialog + run, returns a result, bubbletea resumes. Only `internal/glyphclone`
  imports Glyph. Removes `ModeCloneDialog`/`ModeCloneRun` and their views/tests from
  `internal/tui`.
- Glyph has no file-browser widget; the "file browser" example is `List` + `os.ReadDir`.
  Build a directory picker the same way: `..` entry, dirs only, dotdirs hidden,
  current dir = clone target.
- Deploy log = example `deploylog`: `Spinner` + `Progress` + `Log(io.Reader)` fed from an
  `io.Pipe`.
- `clone.Run` is non-streaming, so add `clone.RunProgress` with a per-repo event callback;
  `Run` becomes a wrapper.
- State is a pure struct tested directly; Glyph is a thin view (feasibility risk 4).
- Toolchain: Glyph needs Go 1.25.1; bump go.mod + CI + release workflow.

## Plan

1. [ ] Toolchain bump + glyph dep.
2. [ ] `clone.RunProgress` + test.
3. [ ] `internal/glyphclone`: dialog (browser + preview + org-subdir toggle) and run (log,
       cancel, retry failures) + tests.
4. [ ] Wire `tui` -> `tea.Exec`; delete old clone modes; update `main`/`build`.
5. [ ] Docs: DESIGN.md clone section, ADR-0005 note.

## Notes

- Skipped on purpose: raw `git clone` stderr in the log (only per-repo status lines);
  hidden-dir toggle; typing a path by hand. Add when asked.
