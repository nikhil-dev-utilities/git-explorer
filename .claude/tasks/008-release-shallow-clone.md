# 008 - Push, merge and release the shallow clone toggle

- status: completed
- last updated: 2026-09-25

## Request

User: "push open pr, wait for CI if any, merge, release" (task 007's branch `feat/shallow-clone`).

## Plan

Same release flow as task 006: squash-merge PR (Conventional Commits title, `feat:` so 0.x -> 0.7.0)
-> release-please PR -> merge it -> tag + GoReleaser.

1. [x] Push `feat/shallow-clone`, open PR `feat(clone): add a shallow clone toggle to the clone dialog`.
2. [x] Wait for CI (build-vet-test, commitlint), squash-merge.
3. [x] Merge the release-please PR, watch GoReleaser, verify release assets.

## Notes
- PR #103 squash-merged (CI green) -> release-please PR #104 (0.7.0) -> tag v0.7.0 -> GoReleaser
  published 4 tarballs + checksums.txt.
- CI on the release-please PR always fails at startup ("workflow file issue", no jobs run); same
  for 0.5.0 and 0.6.0. Pre-existing, did not block the release.
- Release: https://github.com/nikhil-dev-utilities/git-explorer/releases/tag/v0.7.0
