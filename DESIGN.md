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

Each pane is a Glyph `FilterList` (the fuzzy-finder component): a live filter row, an
`n/total` counter (with a spinner while pages are still streaming in) and the list.

- Left pane: Orgs. No repo counts — see ADR-0002.
- Right pane: Repos of the open Org, with `pushed_at` age and state badges.
- Two independently-bordered boxes; the focused pane's border is a distinct colour, so
  focus is legible without reading any text. Both panes always show their facet/sort line.
- Filter syntax is fzf's: plain text fuzzy-matches; `'foo` exact, `^foo` starts with,
  `foo$` ends with, `!foo` not, a space is AND, `|` is OR. Regex is not supported.
- Facets (Org affiliation; Repo archived / forks / visibility), sort, Org pane width,
  Host, clear selection and reload live in one menu, `o`, each row showing its current value.
- Tick all matching is a first-class key: `/tf`, Enter, `a`. That is most of the job.
- The Repo title shows the age of the data (`loaded 4m ago`); see "Loading" below.
- The layout is recomputed every frame from the terminal size, so panes reflow live when
  the window is resized; see "Narrow terminals".

### Interaction

Filtering is **modal** (ADR-0009): press `/` to type into the focused pane's filter; Enter
keeps it and returns to the list, Esc clears it. Outside that mode every key is a verb, so
`Space`, `a`, `x`, `r`, `o` and `?` are bare keys and Tab is free to switch panes. Glyph's
`FilterList` routes typing to one pane only, so the shell owns key routing and drives the
focused pane's `FilterList` (`SetQuery`, `SelectNext`, ...) itself; each modal state
(including filtering) owns a key router pushed on entry and popped on exit.

| Key | Action |
|---|---|
| `/` | filter the focused pane (fzf syntax) |
| `↑` `↓` `j` `k` `^p` `^n`, `PgUp` `PgDn` | move the cursor |
| `Tab` / `Shift-Tab` | switch between the Org and Repo panes; never reloads |
| `Enter` `→` `l` | Orgs: open the Org (already shown: just focus) |
| `Enter` `c` | Repos: clone the ticked Repos |
| `Esc` | clear the pane's filter; with none, Repos back to Orgs |
| `←` `h` | Repos back to Orgs; ticks and filters are kept |
| `Space` | tick the Repo and move down |
| `a` | tick every Repo matching the filter; again to untick them |
| `x` | clear the whole Selection, including ticks the filter hides |
| `r` `F5` | reload the focused pane from the Forge |
| `o` | options: Host, facets, sort, pane width, clear selection, reload |
| `?` `F1` | help |
| `^c` | quit (a bare `q` would quit mid-word) |

While filtering, every printable key is text; `↑ ↓ ^p ^n PgUp PgDn` still move the list,
Enter accepts, Esc clears, `^c` quits. The keys people edit text with (`^h ^i ^m ^[ ^a ^e
^u ^w ^k ^z ^d`) must not be bound in this mode.

### Loading

Orgs load at startup, on host switch, and on reload; focusing the Org pane never loads.
Repos load only when an Org is opened for the first time in a session, or on reload.
Reopening the Org already shown only moves focus, keeping its filter and ticks. Repos are
cached per Host and Org **in memory for the session** and served from there when an Org is
reopened; nothing is written to disk. The title shows the age of the data and `r` / `F5`
refetch, which is the refresh affordance and staleness indicator ADR-0007 requires of any
cache. Ticks for Repos that vanished on reload are dropped.

Opening a *different* Org, or switching Host, would discard ticked Repos, so with ticks
present it first asks: clone them now, discard and continue, or stay (ADR-0005, ADR-0009).
Moving focus between panes never prompts.

### Failure surfaces

Errors are scoped to their blast radius rather than funnelled through one widget.

- **Fatal** — full screen, with the exact command to fix it. Missing `git` or `gh`, or no
  `gh` auth for the active Host. Nothing else in the app works, so nothing else is shown.
  Offers `y` to pick another Host rather than only `^c` to quit. Reserved for a Host the
  user explicitly configured themselves failing auth — a real misconfiguration. When the
  active Host instead came from zero-config discovery (or its last-resort implicit
  `github.com` default), the identical "not authenticated" failure downgrades to
  pane-scoped instead: nothing was misconfigured, there was just nothing to discover yet.
- **Pane-scoped** — inline in the affected pane, *keeping whatever already loaded*. A
  progressive load that dies at page 15 keeps its 1,400 Orgs and offers reload (`r`). Losing them
  to a network blip would be the worst possible response.
- **Transient** — the status line. Rate limits with a retry countdown, and anything that
  resolves itself by waiting.

Empty states are distinguishable from each other and from loading: "no Repos in this
Org", "no matches for `tf-`", and an explicit loading indicator are three different
messages, never one blank pane.

### Narrow terminals

The Org pane is 34 columns by default (the `o` menu cycles 28/34/42/52) and the Repo pane takes
the remainder. As width drops, the Repo row sheds the age column first, then state
badges, keeping the name longest. The Org pane shrinks rather than crushing the Repo
pane below 20 columns. Below 60 columns a "terminal too narrow" card replaces the
layout instead of showing a broken one.

### Short terminals, long lists

Each pane has an explicit height (terminal rows minus the three footer rows) and its
list clips to it, scrolling to keep the cursor visible. That is what makes a Private
Host with thousands of Orgs (the exact case progressive loading is built for) navigable
rather than just loadable, and keeps the footer and bottom borders intact on short
terminals.

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
    └─ pre-flight Outcome check → clone screen (folder browser) → bounded parallel `git clone`
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

The clone screen (built on Glyph, `internal/glyphclone`) asks about the Target and the
org-level directory every time; the Target is the user's hierarchy and we do not invent
levels in it. The Target is chosen in a folder browser (directories only, dotdirs hidden,
`..` always available). `/` opens a prompt to type a path to jump to (`~`, absolute, or
relative to the current folder, prefilled with the current folder); `n` prompts for a new
folder name inside the current one. Both are the same modal prompt and take effect on
Enter; Esc cancels.

A folder that does not exist yet is a valid Target: it is only marked "new folder, created
when cloning" and is created by the clone itself (`git clone` parents are made as needed),
so cancelling the screen never leaves an empty directory behind. Every Repo in the batch
lands under the chosen folder. A path that is a file, or sits under one, is rejected in
the prompt. The preview redraws live as the folder or the org-subdirectory toggle changes.
`s` makes the whole Clone Run shallow (`git clone --depth 1`); like the org-subdirectory
toggle it starts off each launch and is remembered for the rest of the session.

```
Clone 4 repos → ~/src
╭─ Choose target folder ──╮╭─ Preview ────────────────────────╮
│> ../                    ││2 to clone · 1 skipped · 1 conflict│
│  work/                  ││                                  │
│  oss/                   ││+ tf-network                      │
│                         ││+ tf-vpc                          │
│                         ││= tf-dns                          │
│                         ││! tf-modules                      │
╰─────────────────────────╯╰──────────────────────────────────╯
org subdirectory: off · shallow: off · parallelism: 8
enter open · ← up · / go to path · n new folder · tab org subdirectory · s shallow · c clone · esc cancel
```

Pressing `c` switches the same screen to a live log: a spinner, progress bar and one line
per Repo as its clone starts and finishes (`→` cloning, `✓` cloned, `=` skipped, `!`
conflict, `✗` failed). When it finishes the summary offers `r` to retry only the failures.
The log shows per-Repo status lines, not raw `git clone` output.

The clone screen is mounted inside the shell's own Glyph app as a full-screen overlay with
its own modal key router; it hands back the chosen Target and whether a run happened. See
ADR-0008 and docs/glyph-feasibility.md.

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
internal/ui/        Glyph shell: Org and Repo panes, facets, selection, options/host/help
                    cards, key routing. state*.go is terminal-free and tested directly.
internal/glyphclone/  the clone screen mounted in the shell: folder browser, preview,
                    live clone log
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

**No config file is required.** A fresh install with no `hosts:` declared works
without one: rather than always assuming `github.com`, git-explorer reads `gh`'s own
`hosts.yml` (never the credential itself — only the host name and `git_protocol`
fields) to discover which Host(s) the user is actually authenticated to, and uses
those, gh-cli Frontdoor, no default Target so the clone dialog opens empty. This is
what makes a GHE-only user (never logged into `github.com` at all) work out of the
box. Falls back to the implicit `github.com` Host only if gh's own file is missing or
unreadable. An explicit `hosts:` list in config.yaml always wins over discovery,
even one that happens to declare the exact same Host discovery would have found.

**Bootstrapped on first launch.** This one reverses an earlier decision: git-explorer
now creates `$XDG_CONFIG_HOME/git-explorer` (or the `~/.config` fallback) itself if it
doesn't exist, and writes a starter `config.yaml` there — header comment only, no
active settings, so it resolves identically to the file-absent case until edited. An
existing file is never touched. There is still no first-run wizard and no interactive
prompt; this is a directory-and-empty-file bootstrap, not config generation. The
earlier stance ("we never write the file ourselves") was strict user-ownership purity;
in practice the friction of a first-run user not knowing where to even put a config
file outweighed that purity.

```yaml
clone:
  default_target: ~/src          # pre-fills the dialog, always editable
  parallelism: 8

log:
  path: ~/.logs/git-explorer/git-explorer.log
  level: info                    # debug | info | warn | error | off
  max_size_mb: 5

hosts:
  - name: github.com
    frontdoor: gh-cli
    protocol: ssh
  - name: ghe.corp.internal
    frontdoor: gh-cli
    protocol: https
    default_target: ~/work       # per-Host override
```

## Logging

A full-screen Glyph program owns stdout and the terminal — they are the render surface, and a stray
write corrupts the display. A file is therefore the *only* diagnostic channel, not a
convenience. Logging to stderr is refused outright rather than merely discouraged.

**Where.** Neither the binary's directory nor the working directory is an appropriate
place to write. The default is `~/.logs/git-explorer/git-explorer.log` — a fixed
path, not resolved against any XDG env var (unlike the config file's own location).
This has moved twice before: originally `$XDG_STATE_HOME` (strict XDG separation of
state from config), then alongside `config.yaml` under `$XDG_CONFIG_HOME` (one
directory beats XDG purity), now this literal path, by explicit request. The
directory doesn't need to exist ahead of time — `lumberjack` creates it, nested
segments included, the same way it creates the log file itself. Overridden in
precedence order:

```
--log-file <path>  >  GIT_EXPLORER_LOG  >  log.path in config  >  ~/.logs/git-explorer/git-explorer.log
```

**On by default**, at `info`. A user reporting "it hung listing orgs" can attach the file
without being asked to reproduce under a flag — which, for an intermittent TUI bug, is
often a dead end. `--log-level=off` disables it entirely.

**Bounded by size**, keeping recent context and dropping the oldest.
`log/slog` writing to a `lumberjack.Logger` with `MaxSize: 5, MaxBackups: 1,
Compress: false`. This is rotation rather than in-place trimming, so it costs one extra
file on disk and buys away a read-rewrite-realign pass over a file that is open for
append — the only part of this feature with real edge cases. A text handler, not JSON:
the audience is a human pasting it into an issue.

**Never logged, at any level:** the stdout of `gh auth token`, which *is* the credential.
ADR-0004 says we hold no secrets; subprocess output is precisely where we could start by
accident, so that one command's output is suppressed at the exec boundary rather than
filtered later.

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
