# Bitbucket credentials come from the environment, then git's credential store

> Amends [ADR-0004](./0004-git-explorer-never-stores-credentials.md), which allowed a
> token source of "the environment or `gh`" only.

Bitbucket has no `gh` equivalent that owns a secret store. The Bitbucket adapter resolves a
credential at the moment it needs one, in this order:

1. **Environment.** Cloud: `BITBUCKET_EMAIL` + `BITBUCKET_API_TOKEN` (Basic auth). This is
   for CI and for explicit overrides.
2. **git's credential store.** The adapter runs `git credential fill` with
   `protocol=https` and `host=<API host>`. On macOS this reads the keychain through
   `credential.helper osxkeychain`.

If both fail, the result is `ErrKindFatal` with a "not authenticated" message that names
both options.

## Why git's credential store

`git clone` over HTTPS already needs a credential, and git already gets it from its helper
(ADR-0003). Reusing the same store means the user keeps one secret in one place, in a store
we did not build. That is the same reasoning ADR-0004 used for `gh`. `git` is already a hard
prerequisite, so this adds no dependency.

Bitbucket Cloud serves its REST API from `api.bitbucket.org` and git from `bitbucket.org`.
The two keychain entries are therefore separate. This matters because the usernames differ:
the API takes the Atlassian email, and git takes the Bitbucket username (or
`x-bitbucket-api-token-auth`). The password is the same API token in both entries.

We rejected app passwords. Atlassian has retired them in favour of API tokens.

## Consequences

- A second place now exists where a secret transits our process: the stdout of
  `git credential fill`. It gets the same rule as `gh auth token`. It is captured at the
  exec boundary, used, and never logged at any level, and errors never include it.
- `git credential fill` must never prompt. Run it with `GIT_TERMINAL_PROMPT=0` and no
  stdin TTY, so a missing entry fails fast instead of hanging the UI.
- The config file still holds no secret. Nothing in this ADR is written to disk by us.
