# 006 - Push v2 and release

- status: completed
- last updated: 2026-09-20

## Request

User: "lets push and get a release".

## Plan

Release flow (release-please, .github/workflows/release-please.yml): squash-merge a PR to main
-> release-please opens/updates a release PR -> merging it tags and GoReleaser publishes binaries.

1. [x] Rebase v2 onto origin/main (0.5.0 already released; manifest there is 0.5.0).
2. [x] go vet / go test -race / gofmt / linux build before pushing.
3. [x] Push v2, open PR to main. Title must be Conventional Commits (commitlint checks the PR
       title, and squash-merge makes it the commit release-please reads). Use `feat:` not `feat!:`
       and no BREAKING CHANGE footer: in 0.x release-please would otherwise jump to 1.0.0.
       Expected next version: 0.6.0.
4. [x] Wait for CI (build-vet-test, commitlint), squash-merge.
5. [x] Merge the release-please PR, watch GoReleaser, verify the GitHub release assets.

## Notes
- Push was rejected: "email privacy restrictions". Local commits used the Gmail address; main's commits use
  `fernandesnikhil@users.noreply.github.com`. Re-authored the 20 unpublished commits with env vars
  (GIT_AUTHOR_*/GIT_COMMITTER_*, no git config change), tree identical, then pushed.
  Future pushes from this checkout will hit the same block until `git config user.email` is the noreply address.
- PR #101 (squash, title `feat(ui): ...`) -> release-please PR #102 (0.6.0) -> tag v0.6.0 -> GoReleaser
  published 4 tarballs + checksums.txt (darwin/linux x amd64/arm64). CI green on Linux (-race included).
- Release: https://github.com/nikhil-dev-utilities/git-explorer/releases/tag/v0.6.0
