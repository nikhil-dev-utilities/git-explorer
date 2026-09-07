# Org discovery differs by Host type

The Org pane is populated by two different strategies depending on the Host, and this is
deliberate rather than an accident of incremental development.

On a **Private Host** we list *every* Org on the instance (`GET /organizations`),
including Orgs the user has no Affiliation with, and badge each row with its Affiliation.
Discovery across an unfamiliar internal estate is the point — you cannot clone what you
cannot see, and on a corporate instance the Orgs you are not a member of are often
exactly the ones you were sent to find.

On a **Public Host** we do the opposite and list only Orgs where Affiliation is Owner,
Member or Collaborator. github.com's `GET /organizations` enumerates every organization
in existence in id order; it is useless for this purpose. Member Orgs come from
`GET /user/orgs`, and Collaborator Orgs — invisible to that call, and the case that
motivated the requirement — are recovered from a cheap
`GET /user/repos?affiliation=collaborator` probe.

We rejected deriving the Org list from a full `GET /user/repos` sweep on both Hosts. It
would unify the code paths, but it cannot produce zero-Affiliation Orgs at all, so it
fails the Private Host requirement outright, and it forces an expensive up-front fetch to
answer a question the user has not yet asked.

## Consequences

- The Org pane shows **no repo counts**. Repos load lazily per Org, so a count would
  require fetching every Org's repos — the exact cost this design avoids.
- `Affiliation: None` is a normal, displayable state, not an error. Selecting such an Org
  is allowed; it will simply list only the Repos the credentials can see.
