# 009 - Bitbucket support (HTTPS clone + REST API)

- status: completed
- last updated: 2026-10-01 (released v0.8.0)

## Request

User asked whether multiple servers are supported (yes, GitHub-only: one Forge serves every
Host) and how to add Bitbucket with HTTPS clone and REST API. Agreed approach: env vars first,
then `git credential fill`. User: "file amendments, then issues, then present plan for execution".

## Plan

1. [x] Branch `docs/bitbucket-adrs`. Write ADR-0010 (each Host names its Forge, a mux routes)
       and ADR-0011 (Bitbucket credentials: env, then git credential store; amends ADR-0004).
       Add a pointer in ADR-0004 and Workspace mapping in CONTEXT.md. Commit.
2. [x] File GitHub issues: forge routing, gh prereq only for GitHub Hosts, Bitbucket Cloud
       adapter, credentials, docs; Data Center deferred (needs-triage).
3. [x] Present execution plan (issue order, branches, PRs).
4. [x] Docs PR #111 opened (not merged).
5. [x] #105 on `feat/forge-routing`, stacked on `docs/bitbucket-adrs`.
6. [x] #106 gh prereq (PR #115), #107 credentials (PR #116).
7. [x] #108 Cloud adapter (PR #117), #109 docs (PR #118). All merged.
8. [x] README Hosts section (PR #119). Release PR #114 merged on user request, v0.8.0 published.
   Manual check against a real bitbucket.org account was NOT done before release.

## Notes
- ADRs committed on branch `docs/bitbucket-adrs` (not pushed): ADR-0010, ADR-0011, ADR-0004 amended note, CONTEXT.md.
- Issues: #105 forge routing, #106 gh prereq conditional, #107 Bitbucket credentials,
  #108 Bitbucket Cloud adapter, #109 docs, #110 Data Center (needs-triage).
- Execution order: docs PR -> #105 -> (#106, #107 parallel) -> #108 -> #109 -> release.
- #112 auto-closed when its stacked base branch was deleted on merge; reopened as #113.
  Lesson: do not stack PRs on a branch that `--delete-branch` will remove.
- Atlassian removed `GET /2.0/user/permissions/workspaces` (CHANGE-2770). The adapter uses
  `GET /2.0/user/workspaces` instead: only an `administrator` flag, so Affiliation is Owner or Member.
- CloneURL is built by hand. `links.clone` https embeds the API user (an email), which is the wrong git username.
- Release: https://github.com/nikhil-dev-utilities/git-explorer/releases/tag/v0.8.0 (4 tarballs + checksums.txt).
