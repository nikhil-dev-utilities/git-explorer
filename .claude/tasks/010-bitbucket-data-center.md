# 010 - Bitbucket Data Center / Server support (#110)

- status: completed
- last updated: 2026-10-12

## Request

Work on issue #110: support self-hosted Bitbucket Data Center / Server as a Private Host.

## Plan

Decision: keep `forge: bitbucket`. Cloud vs DC is inferred from the host name, like
`hostKind` for GitHub: `bitbucket.org` is Cloud (Public Host), any other name is DC (Private Host).
No new config key.

1. [x] ADR-0012 records the decision, DC credentials, clone URLs.
2. [x] Adapter: per-Host credential cache; DC ListOrgs (projects), ListRepos (project repos),
       start/isLastPage pagination; clone URL taken from `links.clone` at listing time
       (ssh port and context path vary per instance), carried on `forge.Repo`.
3. [x] DC credentials: `BITBUCKET_TOKEN` (Bearer) from env, else `git credential fill` for the host.
4. [x] Tests with httptest.
5. [x] Docs: README, DESIGN.md, bootstrap config comment.
6. [x] PR #120 merged (not released, per user). Not verified against a real DC instance (none available).

## Notes
- Commits: adapter + tests, then ADR-0012 + docs. Branch `feat/bitbucket-data-center`.
- Not covered: personal projects (`~user`), context-path installs, cross-protocol Skipped
  for DC URLs (reports Conflict, safe), Affiliation always None.
- Needs a manual check against a real DC instance before release.
