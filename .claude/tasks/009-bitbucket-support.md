# 009 - Bitbucket support (HTTPS clone + REST API)

- status: in_progress
- last updated: 2026-10-01

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
6. [ ] #106 gh prereq, #107 credentials.
7. [ ] #108 Cloud adapter, #109 docs, release.

## Notes
- ADRs committed on branch `docs/bitbucket-adrs` (not pushed): ADR-0010, ADR-0011, ADR-0004 amended note, CONTEXT.md.
- Issues: #105 forge routing, #106 gh prereq conditional, #107 Bitbucket credentials,
  #108 Bitbucket Cloud adapter, #109 docs, #110 Data Center (needs-triage).
- Execution order: docs PR -> #105 -> (#106, #107 parallel) -> #108 -> #109 -> release.
