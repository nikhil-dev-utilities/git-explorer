# 008 - Push, merge and release the shallow clone toggle

- status: in_progress
- last updated: 2026-09-25

## Request

User: "push open pr, wait for CI if any, merge, release" (task 007's branch `feat/shallow-clone`).

## Plan

Same release flow as task 006: squash-merge PR (Conventional Commits title, `feat:` so 0.x -> 0.7.0)
-> release-please PR -> merge it -> tag + GoReleaser.

1. [ ] Push `feat/shallow-clone`, open PR `feat(clone): add a shallow clone toggle to the clone dialog`.
2. [ ] Wait for CI (build-vet-test, commitlint), squash-merge.
3. [ ] Merge the release-please PR, watch GoReleaser, verify release assets.
