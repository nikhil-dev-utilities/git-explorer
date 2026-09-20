# git-explorer

A terminal tool for finding repositories across many organizations and getting a
chosen set of them onto local disk. Browsing exists to build a **Selection**; cloning
is the terminal action. Everything below the Repo level — branches, files, commits,
issues — is out of scope.

```
╭─ Orgs ─────────────────╮╭─ Repos: acme · 3 selected ─────────────────╮
│ sort: name             ││ archived: hide · forks: hide · sort: name  │
│ > plat                 ││ > tf                                       │
│   2/38                 ││   6/38                                     │
│ > platform-eng  collab ││   [x] tf-network                   2d ago  │
│   platform-ops  none   ││   [x] tf-dns                       1mo ago │
│                        ││ > [x] tf-vpc                       3h ago  │
╰────────────────────────╯╰────────────────────────────────────────────╯
 host: ghe.corp.internal · 3 selected
 type filter · ↑↓ move · tab tick · ^a all · enter clone · esc back · ^o options · F1 help
```

Left pane: Orgs (namespaces that own repos — a personal account counts as one too).
Right pane: Repos of the open Org. Both are fuzzy finders: type to filter the focused
pane (fzf syntax — `'exact`, `^start`, `end$`, `!not`). `^a` ticks everything currently
matching the filter; that one combination is most of the job. The panes reflow live as
you resize the terminal.

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

Start typing to filter Orgs, `Enter` to open one, `Tab` to tick Repos, `Enter` again to
open the clone screen, browse to the target folder, `c` to clone — everything clones into
the current directory by default. `^o` opens the options menu and `F1` shows every key.

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

Typing always filters the focused pane, so actions live on a few non-printing keys. The
filter uses [fzf](https://github.com/junegunn/fzf#search-syntax) syntax: plain text is a
fuzzy match; `'foo` exact, `^foo` starts with, `foo$` ends with, `!foo` not, a space is
AND and `|` is OR. Everything else you might want to change (host, hide archived/forks,
visibility, sort, pane width, reload) is one menu: `^o`. `F1` shows this table in the app.

| Mode | Key | Action |
|---|---|---|
| Browse | type | filter the focused pane (fzf: 'exact ^start end$ !not) |
| Browse | ↑/↓, ^p/^n | move the cursor |
| Browse | PgUp/PgDn | move a page |
| Browse | Enter, → | Orgs: open the Org · Repos: clone the ticked Repos |
| Browse | Esc, ← | Repos: back to Orgs · Orgs: clear the filter |
| Browse | Tab, Shift-Tab | tick the Repo and move down / up |
| Browse | ^a | tick every Repo matching the filter |
| Browse | ^o | options: host, facets, sort, pane width, reload |
| Browse | F1 | this help |
| Browse | ^c | quit |
| LeavePrompt | c | clone now |
| LeavePrompt | d | discard the Selection |
| LeavePrompt | Esc | stay |
| LeavePrompt | ^c | quit |
| CloneDialog | ↑/↓, j/k | move in the folder browser |
| CloneDialog | Enter | open the highlighted folder |
| CloneDialog | ←, h, Backspace | go up a folder |
| CloneDialog | / | type a path to jump to (~, absolute or relative); Enter go · Esc cancel |
| CloneDialog | n | new folder in this one, made when cloning; Enter create · Esc cancel |
| CloneDialog | Tab | toggle the org-subdirectory path |
| CloneDialog | c | start the Clone Run |
| CloneDialog | Esc, ^c | cancel, cloning nothing |
| CloneRun | Esc, ^c | cancel the run (while in flight) |
| CloneRun | r | retry failed Repos only |
| CloneRun | Esc | done: back to Browse |
| Options | ↑/↓ | move |
| Options | Enter, Space | change the value |
| Options | Esc, ^o | close |
| HostSwitch | ↑/↓, ^p/^n | move the cursor |
| HostSwitch | Enter | switch to this Host |
| HostSwitch | Esc | cancel |
| HostSwitch | ^c | quit |
| Fatal | ^y | switch to a different Host |
| Fatal | ^c | quit |

`LeavePrompt` guards a non-empty Selection when you try to leave the Repo pane without
cloning it; `Options` is the `^o` menu; `CloneDialog` is a folder browser that picks the
Target and previews where each Repo will land; `CloneRun` streams the clone itself into a
live log; `HostSwitch` lists every Host from your config; `Fatal` shows when something
(usually `gh` auth) needs fixing before anything else can work.

`internal/ui/readme_test.go` asserts this table against the app's own keymap table, so it
can't silently drift from what the running app actually does.

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
