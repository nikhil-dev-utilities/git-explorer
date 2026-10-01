# 009 - Bitbucket support (HTTPS clone + REST API)

- status: in_progress
- last updated: 2026-10-01

## Request

User asked whether multiple servers are supported (yes, GitHub-only: one Forge serves every
Host) and how to add Bitbucket with HTTPS clone and REST API. Agreed approach: env vars first,
then `git credential fill`. User: "file amendments, then issues, then present plan for execution".

## Plan

1. [ ] Branch `docs/bitbucket-adrs`. Write ADR-0010 (each Host names its Forge, a mux routes)
       and ADR-0011 (Bitbucket credentials: env, then git credential store; amends ADR-0004).
       Add a pointer in ADR-0004 and Workspace mapping in CONTEXT.md. Commit.
2. [ ] File GitHub issues: forge routing, gh prereq only for GitHub Hosts, Bitbucket Cloud
       adapter, credentials, docs; Data Center deferred (needs-triage).
3. [ ] Present execution plan (issue order, branches, PRs).

## Notes
