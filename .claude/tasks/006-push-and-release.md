# 006 - Push v2 and release

- status: in_progress
- last updated: 2026-09-20

## Request

User: "lets push and get a release".

## Plan

Release flow (release-please, .github/workflows/release-please.yml): squash-merge a PR to main
-> release-please opens/updates a release PR -> merging it tags and GoReleaser publishes binaries.

1. [ ] Rebase v2 onto origin/main (0.5.0 already released; manifest there is 0.5.0).
2. [ ] go vet / go test -race / gofmt / linux build before pushing.
3. [ ] Push v2, open PR to main. Title must be Conventional Commits (commitlint checks the PR
       title, and squash-merge makes it the commit release-please reads). Use `feat:` not `feat!:`
       and no BREAKING CHANGE footer: in 0.x release-please would otherwise jump to 1.0.0.
       Expected next version: 0.6.0.
4. [ ] Wait for CI (build-vet-test, commitlint), squash-merge.
5. [ ] Merge the release-please PR, watch GoReleaser, verify the GitHub release assets.

## Notes
