# Each Host names its Forge, and one router Forge serves the UI

Until now one `forge.Forge` (the GitHub adapter) served every Host, so "multiple Hosts" meant
"multiple GitHub installs". Bitbucket is the first second Forge. Each Host in the config now
names its Forge (`forge: github | bitbucket`, default `github`), and `forge.Host` carries it.

The composition root builds one adapter per Forge kind in use and wraps them in a small
router that implements `forge.Forge` itself. It dispatches `ListOrgs` on `host.Forge`, and
`ListRepos` / `CloneURL` on the `Host` already carried by the Org or Repo. The UI keeps one
`Forge` in `ui.Deps` and never learns that more than one exists. ADR-0001 still holds:
`Forge` is the only port, and Frontdoors stay private to each adapter.

We rejected putting a `map[kind]Forge` in `ui.Deps`. It would leak Forge plurality into the
UI for no behavioural gain, because every call already has a Host in hand.

The unused `frontdoor:` config key stays. It still means "how to reach this Forge" and is
validated against the Host's Forge: `gh-cli` for GitHub, `rest` for Bitbucket.

## Bitbucket fits the two-level port

Bitbucket Cloud has Workspace → Project → Repo. We map the Workspace to the Org and ignore
Projects. Repos list at Workspace level, so the Org → Repo port is unchanged. Projects may
become a facet later. Bitbucket Data Center maps Project to Org, which fits as well, but it
is a different API and is deferred until someone needs it.

`bitbucket.org` is a Public Host: only Workspaces with an Affiliation are Visible, which
matches ADR-0002. Bitbucket has no archived flag, so `Archived` is always false there.

## Consequences

- Prerequisite checks depend on the config: `gh` is required only when a GitHub Host is
  configured, so they run after config load.
- Host discovery from `gh` still finds GitHub Hosts only. Bitbucket Hosts are always
  declared in `hosts:`.
- The session Repo cache is already keyed by Host name, so it needs no change.
