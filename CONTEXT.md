# git-explorer

A terminal tool for finding repositories across many organizations and getting a chosen
set of them onto local disk. Browsing exists to build a selection; cloning is the
terminal action.

## Language

### Sources

**Forge**:
A kind of git hosting product with its own domain model — GitHub, GitLab. The Forge is
what the rest of the tool talks to; everything below it is invisible.
_Avoid_: backend, provider, platform, service

**Frontdoor**:
One way of reaching a particular Forge's data — the `gh` CLI, a REST token, GraphQL. A
Frontdoor changes how bytes and credentials move, never what the data means. Frontdoors
belong to a single Forge and are not interchangeable between them.
_Avoid_: transport, driver, client, adapter

**Host**:
A single addressable installation of a git forge, identified by its API endpoint. Two
kinds exist and they behave differently: a **Public Host** (github.com) and a **Private
Host** (a self-managed enterprise install).
_Avoid_: server, instance, endpoint, remote

**Org**:
A namespace on a Host that owns Repos. A user's own personal namespace is modelled as an
Org so the navigation has one uniform shape.
_Avoid_: organization, owner, group, account, namespace

**Repo**:
A single cloneable repository belonging to exactly one Org.
_Avoid_: repository, project

### Access

**Affiliation**:
The relationship between the user's credentials and an Org or Repo. One of: **Owner**,
**Member**, **Collaborator**, or **None**. Affiliation is displayed, not just filtered
on — an Org with Affiliation `None` can still be listed and browsed.
_Avoid_: permission, role, access level, membership

**Visible**:
An Org or Repo the user's credentials can list. On a Private Host every Org is Visible
regardless of Affiliation; on a Public Host only Orgs with a non-`None` Affiliation are
Visible.
_Avoid_: accessible, available, allowed

### Selecting and cloning

**Selection**:
The set of Repos marked for cloning within a single Org. A Selection belongs to one Org
and never spans Orgs — leaving an Org resolves its Selection or abandons it.
_Avoid_: cart, basket, marked, checked, staged

**Target**:
The local directory a Clone Run writes into. A default may be configured, but it is
always the user's to confirm or change, and its existing shape is never rearranged.
_Avoid_: destination, output dir, workspace, root

**Clone Run**:
One execution of a Selection against a Target. Every Repo in it ends in exactly one
Outcome.
_Avoid_: job, batch, operation

**Outcome**:
What a Clone Run did to one Repo. Exactly one of: **Cloned** (written fresh), **Skipped**
(the Target path already holds this same Repo), or **Conflict** (the Target path is
occupied by anything else). A Clone Run never writes into an occupied path.
_Avoid_: status, result, state, error
