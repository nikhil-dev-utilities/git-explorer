# Clone by shelling out to the git binary

Clone Runs exec `git clone` rather than using a pure-Go implementation such as go-git.
This trades away the self-contained-binary property that Go projects usually prize, and
it is a deliberate trade.

Shelling out inherits the user's entire git environment for free: ssh-agent and
`~/.ssh/config`, credential helpers, `url.<base>.insteadOf` rewrites, HTTP proxies and
corporate CA bundles. On a private enterprise Host behind a corporate proxy, that set is
the difference between the tool working on day one and a long tail of authentication bug
reports we would have to reimplement our way out of. go-git does not read `~/.gitconfig`,
so those settings silently do not apply — silently being the operative word, since the
failure looks like a network error rather than a missing feature.

## Consequences

- `git` is a hard runtime dependency, checked at startup with an actionable error rather
  than discovered at the moment a Clone Run fails.
- Clone progress must be parsed from git's stderr instead of arriving as structured
  events.
- The same choice supplies the Outcome check: `git -C <path> remote get-url origin`
  distinguishes Skipped from Conflict.
- Tests exercise real `git` against local bare repositories in `t.TempDir()`, which is
  both faster and more truthful than mocking a clone.
