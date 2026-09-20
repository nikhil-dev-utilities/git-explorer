# 002 - Glyph port, slice 1: clone view

- status: completed
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

1. [x] Toolchain bump + glyph dep.
2. [x] `clone.RunProgress` + test.
3. [x] `internal/glyphclone`: dialog (browser + preview + org-subdir toggle) and run (log,
       cancel, retry failures) + tests.
4. [x] Wire `tui` -> `tea.Exec`; delete old clone modes; update `main`/`build`.
5. [x] Docs: DESIGN.md clone section, ADR-0005 note.

## Notes

- Skipped on purpose: raw `git clone` stderr in the log (only per-repo status lines);
  hidden-dir toggle; typing a path by hand. Add when asked.

## Outcome

- Done: `clone.RunProgress`; `internal/glyphclone` (folder browser, preview, log, retry);
  `tui` launches it via `tea.Exec`; old bubbletea clone dialog/run deleted.
- Verified in a real pty (python `pyte` driver): browse, org-subdir toggle, streaming log,
  failure + retry, Esc back to bubbletea with result, second launch works.
- Gotchas found and fixed (worth remembering for later slices):
  - Glyph `App.Stop()` closes `os.Stdin` -> kills bubbletea's input. Fix: give Glyph a
    private `/dev/tty` handle.
  - On macOS, closing that handle does not wake Glyph's blocked read (needs another key).
    Fix: `TIOCSTI` NUL in `wake_darwin.go`. Linux path (epoll wakes on Close) is untested.
  - A pipe as Glyph's stdin hangs `QueryDefaultColors` (needs real tty read timeouts).
  - Glyph named views' templates are not exposed, so headless tests need a single view
    with `If`/`Else` and `VBox.Grow(1)` at every level.
- Not done (by choice): raw `git clone` output in the log, hidden-dir toggle, typing a
  path by hand, Linux verification of the stop path.
