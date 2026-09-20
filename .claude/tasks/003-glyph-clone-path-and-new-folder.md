# 003 - Clone screen: type-a-path jump and new folder

- status: completed
- last updated: 2026-09-20

## Request

Add to the Glyph clone screen: (1) type a path the folder browser jumps to, faster than
pane navigation; (2) a way to create a new directory at the selected path so all clones in
the batch land under it.

## Design (decided)

- One modal prompt row, two entry keys: `/` go to path (prefilled with current dir; `~`,
  absolute, or relative to current dir) and `n` new folder (relative name, may be nested).
- New folder is virtual: nothing is written until the Clone Run. `performClone` already
  MkdirAlls the parent, so navigating into a non-existent dir is enough. Header shows
  "(new folder, created when cloning)". Esc-cancelling leaves no empty dir behind.
- A typed path that does not exist is treated the same as `n` (new folder). A path that is
  a file, or lies under a file, is rejected in the prompt.
- Prompt is a pushed riffkey router with `TextInput`, so dialog keys (c, h, j...) type into
  it instead of firing.

## Plan

1. [x] state: prompt, jump, resolve, dirNote + tests.
2. [x] view: prompt row, notice, hints; wire pushed router + tests.
3. [x] pty check with pyte driver.
4. [x] README/help table/DESIGN update; commit.

## Notes

- Skipped on purpose: Tab completion of the typed path, path history. Add when asked.

## Outcome

- Done: `/` goto and `n` new-folder prompt, virtual new folder, rejects files, batch lands
  under the chosen folder. Verified in a pty.
- Gotcha: riffkey treats digits 1-9 as vim count prefixes and swallowed them from the text
  prompt; the prompt router needs `.NoCounts()` (regression test in view_test.go).
- Note: the `TextInput` cursor is a byte index; non-ASCII paths untested.
