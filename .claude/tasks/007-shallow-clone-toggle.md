# 007 - Shallow clone toggle

- status: completed
- last updated: 2026-09-25

## Request

User: "I need to add an option to allow shallow clones of selected repos".

Decisions (asked): run-wide toggle in the clone dialog, fixed `--depth 1`, no config key,
remembered for the session like the org-subdirectory toggle.

## Plan

1. [x] `clone.Repo.Shallow` -> `performClone` adds `--depth 1`. Retry keeps it (Result.Repo).
       Test against a local bare repo: shallow clone has 1 commit.
2. [x] Clone dialog: `s` toggles shallow; shown in options line; Request/Response carry it;
       startRun stamps it onto each Repo. Shell remembers it (internal/ui/state.go).
3. [x] Keymap table, README keybindings table, DESIGN.md dialog mock.
4. [x] go vet, go test -race, gofmt.
