# git-explorer never stores credentials

The config file holds only non-secret facts — hostnames, Frontdoor choice, clone
protocol, default Target paths. It is safe to commit to a dotfiles repo and safe to paste
into a bug report. Credentials are never read from it, written to it, or cached anywhere
by us.

v1 ships a single Frontdoor, `gh-cli`, and asks `gh auth token --hostname X` for a
credential at the moment it is needed. `gh` already owns the secret store — the macOS
keychain on darwin — and already handles enterprise Host login. We are not going to build
a second, worse one.

We rejected reading `~/.config/gh/hosts.yml` directly to auto-discover Hosts: it is gh's
private file format with no compatibility promise, and it would put live tokens in our
process for no benefit. Hosts are declared in our own config instead, which is a few
lines of YAML and cannot break when gh cuts a release.

## Consequences

- `gh` is a hard runtime dependency alongside `git`, checked at startup.
- A future REST or GraphQL Frontdoor must source its token from the environment or from
  `gh`, never from our config. If that ever becomes untenable, this ADR is the thing to
  revisit deliberately — not to work around.
- No log redaction, file-permission enforcement, or secret-scrubbing in panic handlers is
  needed, because there is no secret in our process to leak.
