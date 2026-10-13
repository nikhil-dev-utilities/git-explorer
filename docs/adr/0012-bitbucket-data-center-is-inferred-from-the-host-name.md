# Bitbucket Data Center is inferred from the Host name

> Builds on [ADR-0010](./0010-each-host-names-its-forge.md) and
> [ADR-0011](./0011-bitbucket-credentials-from-env-or-git-credential-store.md).

A Bitbucket Host named `bitbucket.org` is Bitbucket Cloud. Any other Bitbucket Host is
Bitbucket Data Center (or Server). There is no `forge: bitbucket-dc` and no new config key.
This is the same rule `hostKind` already applies to GitHub: `github.com` is the Public Host,
and every other name is a Private Host. One `bitbucket` adapter serves both and branches on
the Host.

We rejected a separate `bitbucket-dc` Forge. The user would have to state something the
Host name already says, and the two would share the credential and error code anyway.

## How Data Center maps

- **Org** is the Project, listed with `GET /rest/api/1.0/projects`. The Org name is the
  Project key. A Private Host lists every Visible Org (ADR-0002). Data Center has no cheap
  per-Project membership call, so Affiliation is `None`. Personal projects (`~user`) are
  not listed.
- **Repo** comes from `GET /rest/api/1.0/projects/{key}/repos`. Pages follow `start`,
  `limit` and `isLastPage`. Data Center reports no push time, so `PushedAt` is zero.
- **Clone URL** is the `links.clone` entry listed with the Repo, carried on
  `forge.Repo.CloneLink`. It is not built, because the ssh port and the context path differ
  per instance. The user in the http link is dropped, so git asks its helper as usual. If
  ssh is disabled, the http link is used.
- The REST API is assumed at `https://<host>/rest/api/1.0`. An instance served under a
  context path is not supported yet.

## Credentials

The same host serves REST and git, so one `git credential fill` entry (username plus HTTP
access token as password) covers both. The order of ADR-0011 holds:

1. **Environment:** `BITBUCKET_TOKEN`, sent as a Bearer token. The Cloud variables are
   never sent to a Data Center Host.
2. **git's credential store** for `host=<host>`, sent as Basic auth.

Credentials are now cached per Host, so a Cloud Host and a Data Center Host can be used
in one session.

## Consequences

- No real Data Center instance was available. The adapter is tested against recorded
  response shapes only.
- A Skipped check compares clone URLs by host, owner and name. A Data Center https URL
  (`/scm/key/repo`) and ssh URL (`:7999/key/repo`) do not compare equal, so an existing
  clone made over the other protocol is reported as a Conflict. That is the safe direction.
