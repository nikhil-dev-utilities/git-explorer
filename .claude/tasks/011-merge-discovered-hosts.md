# 011 - Merge gh-discovered Hosts with the hosts: list (#122)

- status: in_progress
- last updated: 2026-10-12

## Request

Keep the default GHE Host (found through `gh` discovery) when the user adds more Hosts,
such as Bitbucket, in `hosts:`. User asked whether to write the defaults into the config
on start. Agreed instead: merge discovery with the list. User: "file an issue and build it".

## Plan

1. [x] Issue #122.
2. [x] Config: `gh_discovery` key, default true.
3. [x] main.go: discovered Hosts first, `hosts:` entries override by name, then append.
       `gh_discovery: false` keeps the exact list.
4. [x] ui.Deps: replace `HostsUserConfigured` with per-Host `ConfiguredHosts`.
5. [x] Tests.
6. [x] ADR-0013, README, DESIGN.md, starter config.
7. [ ] PR, merge on user approval. No release.
