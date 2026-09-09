# git-explorer

A terminal tool for finding repositories across many organizations and getting a
chosen set of them onto local disk. Browsing exists to build a **Selection**; cloning
is the terminal action. Everything below the Repo level — branches, files, commits,
issues — is out of scope.

```
┌─ Orgs ─────────────────┬─ Repos: acme ───────────────────────────┐
│ plat█                  │ tf-                            38 → 6   │
│ affiliation: any       │ archived: hide · forks: hide · vis: all │
│────────────────────────│─────────────────────────────────────────│
│ acme          member   │ [x] tf-network              2d ago      │
│ platform-eng  collab   │ [x] tf-dns                  1mo ago     │
│ platform-ops  —      > │ [x] tf-vpc                  3h ago      │
│ globex        owner    │ [ ] tf-modules   archived   1y ago      │
└────────────────────────┴─────────────────────────────────────────┘
 host: ghe.corp.internal · 3 selected · ^y host · enter clone · F1 help
```

Left pane: Orgs (namespaces that own repos — a personal account counts as one too),
filtered by name and Affiliation. Right pane: Repos of the focused Org. Filter either
pane by typing — the filter box is always focused, fzf-style — and a leading `/`
switches it to regex. `^o` (or `alt-a`) selects everything currently matching the
filter; that one combination is most of the job.

## Install

### Download a binary

Prebuilt binaries for darwin and linux (amd64 and arm64) are published on the
[Releases page](https://github.com/nikhil-dev-utilities/git-explorer/releases).
Download the archive for your platform, extract it, and put the `git-explorer` binary
somewhere on your `PATH`.

### From source

```sh
go install github.com/nikhil-dev-utilities/git-explorer/cmd/git-explorer@latest
```

### Prerequisites

Either way, you need `git` and the [GitHub CLI](https://cli.github.com) (`gh`) on your
`PATH`, and `gh` authenticated (`gh auth login`) against whichever Host you plan to
browse. git-explorer checks for both at startup and tells you exactly what's missing
rather than failing partway through.

## Quick start

With no config file at all, git-explorer works straight away against whatever
Host(s) `gh` is already logged into — `github.com`, a GitHub Enterprise instance, or
both:

```sh
git-explorer
```

Start typing to filter Orgs, `Enter` to descend into one, `Tab` to tick Repos, `Enter`
again to open the clone dialog, `Enter` once more to confirm — everything clones into
the current directory by default. `F1` shows the full keybinding list at any time.

### Configuration (optional)

git-explorer creates `$XDG_CONFIG_HOME/git-explorer` (or `~/.config/git-explorer`)
on first launch if it doesn't exist yet, along with an empty starter `config.yaml`
there — it's inert until you edit it. Uncomment or add settings to set a default
clone target, add additional Hosts (including self-managed GitHub Enterprise
installs), and tune parallelism and logging. See [DESIGN.md](./DESIGN.md#config) for
the full schema. `--config <path>` and `--log-file <path>` override it from the
command line.

The log file lives at `~/.logs/git-explorer/git-explorer.log` by default, rotated
by size — `--log-file <path>` or `GIT_EXPLORER_LOG` override it.

## Keybindings

The ctrl keymap is the documented baseline and works in a stock terminal with no
configuration. The `alt-` aliases are a bonus for terminals that send Meta for
Option — never a requirement, and never the *only* way to reach an action. `F1` shows
this same table inside the app.

| Mode | Key | Alt | Action |
|---|---|---|---|
| Browse | type |  | edit the focused pane's filter |
| Browse | / |  | switch the filter to regex |
| Browse | ↑/↓, ^p/^n |  | move the cursor |
| Browse | → |  | Orgs: descend (same as Enter) |
| Browse | ← |  | Repos: back to Orgs (same as Esc) |
| Browse | Enter | alt-c | Orgs: descend · Repos: open clone dialog |
| Browse | Esc |  | Repos: back to Orgs · Orgs: clear filter |
| Browse | Tab |  | tick/untick the focused Repo |
| Browse | ^o | alt-a | select all Repos matching the filter |
| Browse | ^t | alt-x | Orgs: cycle Affiliation · Repos: cycle archived |
| Browse | ^f | alt-f | Repos: cycle fork |
| Browse | ^v | alt-v | Repos: cycle visibility |
| Browse | ^s | alt-s | cycle sort: name ⇄ last activity |
| Browse | ^y | alt-h | switch Host |
| Browse | ^r | alt-r | retry a pane-scoped load failure |
| Browse | F1 |  | this help screen |
| Browse | ^c |  | quit |
| LeavePrompt | c |  | clone now |
| LeavePrompt | d |  | discard the Selection |
| LeavePrompt | Esc |  | stay |
| LeavePrompt | ^c |  | quit |
| CloneDialog | type |  | edit the target path |
| CloneDialog | Tab |  | toggle the org-subdirectory path |
| CloneDialog | Enter |  | confirm and start the Clone Run |
| CloneDialog | Esc |  | cancel, cloning nothing |
| CloneDialog | ^c |  | quit |
| CloneRun | ^c |  | cancel the run (while in flight) · quit (once done) |
| CloneRun | r |  | retry failed Repos only |
| CloneRun | Esc |  | done — back to Browse |
| HostSwitch | ↑/↓, ^p/^n |  | move the cursor |
| HostSwitch | Enter |  | switch to this Host |
| HostSwitch | Esc |  | cancel |
| HostSwitch | ^c |  | quit |
| Fatal | ^y | alt-h | switch to a different Host |
| Fatal | ^c |  | quit |

`LeavePrompt` guards a non-empty Selection when you try to leave the Repo pane without
cloning it; `CloneDialog` previews where each Repo will land before you confirm;
`CloneRun` is the clone itself; `HostSwitch` lists every Host from your config;
`Fatal` takes over the screen when something (usually `gh` auth) needs fixing before
anything else can work.

`internal/tui/readme_test.go` asserts this table against the app's own keymap table,
so it can't silently drift from what the running app actually does.

## Vocabulary

git-explorer uses a small, precise vocabulary — **Forge**, **Host**, **Org**, **Repo**,
**Affiliation**, **Selection**, **Target**, **Clone Run**, **Outcome** — defined in
full in [CONTEXT.md](./CONTEXT.md). The short version: a **Host** is one addressable
Forge installation (`github.com`, or a self-managed enterprise install); an **Org** is
a namespace on a Host that owns Repos; a **Selection** is the set of Repos you've
ticked within one Org, cloned together in a single **Clone Run** against a **Target**
directory.

## Design

The full design rationale — scope, failure surfaces, config schema, and the decisions
behind them — lives in [DESIGN.md](./DESIGN.md) and [docs/adr/](./docs/adr/).
