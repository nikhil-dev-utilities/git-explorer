# git-explorer — design

Terminal tool for finding repositories across many organizations and getting a chosen set
of them onto local disk.

Vocabulary is defined in [CONTEXT.md](./CONTEXT.md) and is used precisely throughout this
document. Decisions that are hard to reverse live in [docs/adr/](./docs/adr/).

## What it is for

Bulk clone. Browsing exists to build a Selection; cloning is the terminal action.
Everything below the Repo level — branches, files, commits, issues — is out of scope.
Two fixed columns, always.

## Screen

```
┌─ Orgs ─────────────────┬─ Repos: acme ───────────────────────────┐
│ / plat                 │ / tf-                          38 → 6   │
│ affiliation: any       │ archived: hide · forks: hide · vis: all │
│────────────────────────│─────────────────────────────────────────│
│ acme          member   │ [x] tf-network              2d ago      │
│ platform-eng  collab   │ [x] tf-dns                  1mo ago     │
│ platform-ops  —      > │ [x] tf-vpc                  3h ago      │
│ globex        owner    │ [ ] tf-modules   archived   1y ago      │
└────────────────────────┴─────────────────────────────────────────┘
 host: ghe.corp.internal · [h] switch · 3 selected · [c] clone · [?] help
```

- Left pane: Orgs, filtered by name and Affiliation. No repo counts — see ADR-0002.
- Right pane: Repos of the focused Org, with `pushed_at` and state badges.
- Sort: name or last activity, in either pane.
- Filter: substring, case-insensitive. A leading `/` switches the box to regex.
  Starts-with and ends-with are deliberately absent — `^foo` and `foo$` cover them.
- `select all matching` is a first-class key. Filter to `tf-`, hit it, done. That one
  combination is most of the job.

## Data flow

```
  startup
    ├─ check `git` on PATH        ─┐ actionable error, not a late failure
    ├─ check `gh`  on PATH        ─┘
    └─ read config → Host list, default Targets

  Org pane          progressive: pages stream in, navigable at ~100ms
    ├─ Private Host: GET /organizations          (every Org on the instance)
    │                + /user/orgs                (badge: member/owner)
    │                + /user/repos?affiliation=collaborator (badge: collab)
    └─ Public  Host: /user/orgs + the collaborator probe only

  Repo pane         lazy, on Org selection
    └─ GET /orgs/{org}/repos → name, pushed_at, archived, fork, visibility

  Clone Run         modal, one at a time
    └─ pre-flight Outcome check → confirm dialog → bounded parallel `git clone`
```

Nothing is written to disk between runs. No cache, therefore no invalidation, no
staleness, no "why is it showing a repo I deleted". The cost — a full refetch each launch
— is paid down by progressive loading rather than by a cache.

## Clone

Pre-flight, every selected Repo resolves to exactly one Outcome, shown *before* you
commit to the run:

| Outcome | Condition | Action |
|---|---|---|
| **Cloned** | Target path absent | `git clone` |
| **Skipped** | path is a git repo whose `origin` is this Repo | leave it entirely alone |
| **Conflict** | path occupied by anything else | refuse, report separately |

A Clone Run never writes into an occupied path and never aborts on first failure. It
completes, then reports per-Repo Outcomes with failures retryable.

The dialog asks about the org-level directory every time and remembers nothing — the
Target is the user's hierarchy and we do not invent levels in it. The path list redraws
live as the toggle changes.

```
┌─ Clone 4 repos ─────────────────────────┐
│ target: ~/src                           │   ← pre-filled from config, editable
│ [ ] put clones under an acme/ dir       │   ← always starts unticked
│                                         │
│ + acme/tf-network  → ~/src/tf-network   │
│ + acme/tf-vpc      → ~/src/tf-vpc       │
│ = acme/tf-dns      already cloned, skip │
│ ! acme/tf-modules  path in use by       │
│                    globex/tf-modules    │
│                                         │
│ 2 to clone · 1 skipped · 1 conflict     │
│ [enter] clone  [space] toggle  [esc]    │
└─────────────────────────────────────────┘
```

Execution shells out to `git` (ADR-0003), bounded at 8 parallel by default. Protocol is
per-Host config, `ssh` by default, seeded from `gh config get git_protocol` when the
gh-cli Frontdoor is active.

## Selection

A Selection belongs to exactly one Org and never spans Orgs. Leaving an Org with a
non-empty Selection prompts: clone now / discard / stay. There is no cross-Org cart, and
therefore no ticked-but-off-screen state to lose track of.

The cost, accepted knowingly: a Clone Run is modal, so "clone now" on the way out of an
Org means waiting for it before reaching the next one.

## Architecture

```
internal/forge/
  forge.go          the ONLY port the TUI sees
                      ListOrgs(ctx, Host)  → stream of []Org
                      ListRepos(ctx, Org)  → []Repo
                      CloneURL(Repo)       → string
  github/
    github.go       implements forge.Forge
    frontdoor.go    unexported interface — see ADR-0001
    fd_ghcli.go     v1 ships this one only
internal/clone/     pre-flight Outcome check + parallel `git clone`
internal/config/    hostnames, protocol, default Targets. No secrets — ADR-0004.
internal/tui/       bubbletea model, two panes, filter, dialogs
```

GitLab is deferred and may deserve its own model entirely; the port is shaped honestly on
GitHub's two levels rather than generalised on spec (ADR-0001).

## Config

Non-secret only, so it is safe in a dotfiles repo. See ADR-0004.

**Location:** `$XDG_CONFIG_HOME/git-explorer/config.yaml`, falling back to
`~/.config/git-explorer/config.yaml`. `--config <path>` overrides. macOS is not natively
XDG, but `gh` already puts its config at `~/.config/gh`, and sitting next to the tool we
depend on beats matching platform convention.

**Format:** YAML, for the same reason — a user configuring a Host here has almost
certainly just configured one in `gh`'s YAML, and the shape is nested (a list of Hosts
with per-Host overrides) which reads better in YAML than in TOML.

**No config file is required.** A fresh install with no file works: an implicit
`github.com` Host over the gh-cli Frontdoor, `ssh` protocol, no default Target so the
clone dialog opens empty. There is no first-run wizard and we never write the file
ourselves — config is something the user owns, like their Target directory.

```yaml
clone:
  default_target: ~/src          # pre-fills the dialog, always editable
  parallelism: 8

hosts:
  - name: github.com
    frontdoor: gh-cli
    protocol: ssh
  - name: ghe.corp.internal
    frontdoor: gh-cli
    protocol: https
    default_target: ~/work       # per-Host override
```

## Testing

Hermetic. Nothing in CI touches the network or needs a secret, so fork PRs work and the
gate never goes red for a reason unrelated to the change.

| Layer | Approach |
|---|---|
| GitHub adapter | injected command runner, canned `gh api` JSON fixtures |
| Clone engine | **real** `git` against local bare repos in `t.TempDir()` |
| TUI | `teatest` golden files driven by synthetic key events |
| Filter / sort | plain table tests |

Known and accepted gap: GitHub or `gh` changing an output shape is discovered from a user
report, not from CI.

## CI and release

- **PR gate:** build, vet, test, plus a Conventional Commits lint — release-please makes
  commit messages load-bearing, so a malformed one must fail immediately rather than
  produce a wrong version months later.
- **Release:** release-please maintains a release PR with the computed version and a
  generated CHANGELOG; merging it tags, which fires GoReleaser.
- **Targets:** darwin and linux × amd64 and arm64. Windows when someone asks — shipping an
  untested binary is worse than not shipping one.
- **Versioning:** starts at 0.1.0. Pre-1.0, a breaking change to flags or config format
  bumps the minor. There is no library API to break.

## Deliberately out of scope for v1

- Anything below Repo level, and any read-only browsing affordance (README, languages,
  topics, open in browser)
- Fetching or updating existing clones — a Skipped Repo is left strictly alone
- Cross-Org Selections, background or concurrent Clone Runs
- On-disk caching
- Language filter
- GitLab, and the REST/GraphQL Frontdoors
