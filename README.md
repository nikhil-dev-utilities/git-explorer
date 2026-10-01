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
 / filter · space tick · a all · x clear · enter clone · tab orgs · r reload · o options · ? help
```

Left pane: Orgs (namespaces that own repos — a personal account counts as one too).
Right pane: Repos of the open Org. Both are fuzzy finders: press `/` to filter the focused
pane (fzf syntax — `'exact`, `^start`, `end$`, `!not`). `a` ticks everything currently
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

Either way, you need `git` on your `PATH`. For GitHub Hosts you also need the
[GitHub CLI](https://cli.github.com) (`gh`), authenticated (`gh auth login`) against
whichever Host you plan to browse. A config with only Bitbucket Hosts does not need `gh`.
git-explorer checks at startup and tells you exactly what's missing rather than failing
partway through.

## Quick start

With no config file at all, git-explorer works straight away against whatever
Host(s) `gh` is already logged into — `github.com`, a GitHub Enterprise instance, or
both:

```sh
git-explorer
```

Press `/` and type to filter Orgs, `Enter` to keep the filter and `Enter` again to open an Org,
`Space` to tick Repos, `Enter` to open the clone screen, browse to the target folder, `c` to
clone — everything clones into the current directory by default. `o` opens the options menu
and `?` shows every key.

### Configuration (optional)

git-explorer creates `$XDG_CONFIG_HOME/git-explorer` (or `~/.config/git-explorer`)
on first launch if it doesn't exist yet, along with an empty starter `config.yaml`
there — it's inert until you edit it. Uncomment or add settings to set a default
clone target, choose which Hosts to browse (see [Hosts](#hosts-servers) below), and tune
parallelism and logging. See [DESIGN.md](./DESIGN.md#config) for the full schema.
`--config <path>` and `--log-file <path>` override it from the command line.

### Hosts (servers)

A Host is one server git-explorer browses: `github.com`, a GitHub Enterprise install, or
`bitbucket.org`. Hosts are listed in the config file:

- `~/.config/git-explorer/config.yaml`
- or `$XDG_CONFIG_HOME/git-explorer/config.yaml` if you set `XDG_CONFIG_HOME`
- or any file passed with `--config <path>`

**With no `hosts:` key**, git-explorer uses every GitHub Host that `gh` is logged into.
Run `gh auth login --hostname <host>` to add one.

**With a `hosts:` key**, git-explorer uses exactly that list and nothing else.
Auto-detection is off, so list every Host you want, including the ones `gh` used to find:

```yaml
hosts:
  - name: github.com                # forge: github is the default
    protocol: ssh                   # ssh (default) | https
  - name: ghe.corp.internal         # GitHub Enterprise; run `gh auth login --hostname ghe.corp.internal` first
    protocol: https
    default_target: ~/work          # optional: clone folder for this Host only
  - name: bitbucket.org             # Bitbucket Cloud, see below
    forge: bitbucket
    protocol: https
```

The first Host in the list opens at startup. To switch, press `o` → **Host**, choose one,
then press Enter. If you have Repos ticked, git-explorer asks first, because switching drops
them. If a Host isn't authenticated, the error screen offers `y` to switch to another one.

Credentials never go in this file:
- **GitHub Hosts** use `gh`.
- **Bitbucket** uses env vars or git's credential store (see below).

#### Bitbucket Cloud

Add this entry to your `hosts:` list:

```yaml
  - name: bitbucket.org
    forge: bitbucket
    protocol: https     # or ssh
```

Each Bitbucket Workspace you can access shows up as an Org. Its Projects are not shown.

**Credentials** never go in the config file ([ADR-0011](./docs/adr/0011-bitbucket-credentials-from-env-or-git-credential-store.md)).

1. Create an Atlassian API token with the `read:workspace:bitbucket` and
   `read:repository:bitbucket` scopes. Bitbucket app passwords are retired.
2. Give it to git-explorer in one of two ways:
   - **Environment:** set `BITBUCKET_EMAIL` (your Atlassian email) and `BITBUCKET_API_TOKEN`.
   - **git's credential store** (recommended, keeps the token in the macOS keychain):

     ```sh
     git config --global credential.helper osxkeychain
     printf 'protocol=https\nhost=api.bitbucket.org\nusername=<atlassian email>\npassword=<api token>\n\n' \
       | git credential approve
     ```

     The environment wins when both are set.
3. For HTTPS clones, git asks the same helper for a credential for `bitbucket.org`. That
   entry is separate, and its username is your Bitbucket username (or
   `x-bitbucket-api-token-auth`), not your email. The password is the same API token. git
   prompts and saves it on the first clone, or store it up front with the command above
   using `host=bitbucket.org`.

Bitbucket Data Center is not supported yet ([#110](https://github.com/nikhil-dev-utilities/git-explorer/issues/110)).

The log file lives at `~/.logs/git-explorer/git-explorer.log` by default, rotated
by size — `--log-file <path>` or `GIT_EXPLORER_LOG` override it.

## Keybindings

Press `/` to filter the focused pane; until you press Enter (keep the filter) or Esc (clear it),
every key is text. Everywhere else the keys are plain: `Space` ticks a Repo, `a` ticks all matching,
`x` clears the Selection, `Tab` switches panes, `r`/`F5` reloads, `o` opens the options menu, `?`
shows this table. The filter uses [fzf](https://github.com/junegunn/fzf#search-syntax) syntax: plain
text fuzzy-matches; `'foo` exact, `^foo` starts with, `foo$` ends with, `!foo` not, a space is AND
and `|` is OR.

Repos are fetched only when you open an Org for the first time this session, or when you reload.
Switching panes, or opening an Org you already viewed, never refetches; the Repo title shows how old
the data is (`loaded 4m ago`).

| Mode | Key | Action |
|---|---|---|
| Browse | / | filter the focused pane (fzf: 'exact ^start end$ !not, space = AND) |
| Browse | ↑/↓, j/k, ^p/^n | move the cursor |
| Browse | PgUp/PgDn | move a page |
| Browse | Tab, Shift-Tab | switch between the Org and Repo panes (never reloads) |
| Browse | Enter, →, l | Orgs: open the Org (the Org already shown is not refetched) |
| Browse | Enter, c | Repos: clone the ticked Repos |
| Browse | Esc | clear the pane's filter; with none, Repos back to Orgs |
| Browse | ←, h | Repos back to Orgs (ticks and filter are kept) |
| Browse | Space | tick the Repo and move down |
| Browse | a | tick every Repo matching the filter; again to untick them |
| Browse | x | clear the whole Selection, including ticks the filter hides |
| Browse | r, F5 | reload the focused pane from the Forge |
| Browse | o | options: host, facets, sort, pane width, clear selection, reload |
| Browse | ?, F1 | this help |
| Browse | ^c | quit |
| Filter | type | edit the focused pane's filter (every key is text) |
| Filter | ↑/↓, ^p/^n, PgUp/PgDn | move the list while typing |
| Filter | Enter | accept: back to the list, filter kept |
| Filter | Esc | clear the filter and go back to the list |
| Filter | ^c | quit |
| LeavePrompt | c | clone the ticked Repos now |
| LeavePrompt | d | discard the Selection and continue |
| LeavePrompt | Esc | stay |
| LeavePrompt | ^c | quit |
| CloneDialog | ↑/↓, j/k | move in the folder browser |
| CloneDialog | Enter | open the highlighted folder |
| CloneDialog | ←, h, Backspace | go up a folder |
| CloneDialog | / | type a path to jump to (~, absolute or relative); Enter go · Esc cancel |
| CloneDialog | n | new folder in this one, made when cloning; Enter create · Esc cancel |
| CloneDialog | Tab | toggle the org-subdirectory path |
| CloneDialog | s | toggle shallow clones (--depth 1) |
| CloneDialog | c | start the Clone Run |
| CloneDialog | Esc, ^c | cancel, cloning nothing |
| CloneRun | Esc, ^c | cancel the run (while in flight) |
| CloneRun | r | retry failed Repos only |
| CloneRun | Esc | done: back to Browse |
| Options | ↑/↓ | move |
| Options | Enter, Space | change the value |
| Options | Esc, o | close |
| HostSwitch | ↑/↓, ^p/^n | move the cursor |
| HostSwitch | Enter | switch to this Host |
| HostSwitch | Esc | cancel |
| HostSwitch | ^c | quit |
| Fatal | y | switch to a different Host |
| Fatal | ^c | quit |

`Filter` is the mode after `/`. `LeavePrompt` appears when opening another Org (or switching Host)
would discard ticked Repos. `Options` is the `o` menu; `CloneDialog` is a folder browser that picks
the Target and previews where each Repo will land; `CloneRun` streams the clone itself into a live
log; `HostSwitch` lists every Host from your config; `Fatal` shows when something (usually `gh` auth)
needs fixing before anything else can work.

`internal/ui/keymap_test.go` asserts this table against the app's own keymap table, so it can't
silently drift from what the running app actually does.

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
