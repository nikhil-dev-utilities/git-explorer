# Glyph is the only UI, and the keymap shrinks to a few keys plus one options menu

The Bubble Tea UI is replaced end to end by a Glyph app (`internal/ui`, with the clone
screen in `internal/glyphclone`). bubbletea, lipgloss and teatest are gone. The two
frameworks cannot share a terminal, and the Org/Repo view is the long-lived main screen,
so the swap could not be done one screen at a time without a permanent bridge; the clone
screen alone needed `/dev/tty` and `TIOCSTI` workarounds just to hand the terminal back.

Each pane is Glyph's `FilterList` (the fuzzy-finder component). Two consequences are worth
knowing before touching this code:

- **Regex filtering is dropped.** `FilterList` matches with fzf's engine, so the filter
  syntax is fzf's (`'exact`, `^start`, `end$`, `!not`, space = AND, `|` = OR). That covers
  starts-with/ends-with/contains from the original requirements.
- **Glyph routes typing to one `FilterList` only** (the template keeps a single text
  binding; the last list wins) and registers `^n ^p ^d ^u` on the shared router. The shell
  therefore re-registers every key after `SetView`, focus-aware, and drives the focused
  pane through `FilterList`'s programmatic API. Keep the panes at the root of the view: a
  `FilterList` inside an `If` branch is wired through a child scope that sits ahead of that
  override.

## The keymap

> The bindings in this section were revised by [ADR-0009](./0009-modal-filter-and-on-demand-repo-loading.md)
> (modal filter, bare-key verbs). The Glyph decision and the routing notes above still apply.

The old keymap (`^t ^f ^v ^s ^g ^y ^r` plus `alt-` aliases for each) was judged
non-intuitive. ADR-0006's core rule stands: the filter is always live, so no verb is a bare
letter. What changes is how many verbs need a key at all: host, archived, forks,
visibility, sort, Org pane width and reload are rows of one options menu (`^o`) that shows
each row's current value. What remains is arrows, Enter/→, Esc/←, Tab (tick and advance),
`^a` (tick all matching), `^o`, F1 and `^c`.

`^a` is now bound. ADR-0006 listed it as readline "home"; `Home` still does that, and tick-all
is the most common multi-select action. The other forbidden keys (`^h ^i ^m ^[ ^e ^u ^w ^k
^z ^d`) stay forbidden. There are no `alt-` aliases: they never worked in stock macOS
terminals, and one menu replaces the chords they aliased.

## Consequences

- Glyph is pre-1.0 and pinned at v0.8.0. All Glyph imports live in `internal/ui` and
  `internal/glyphclone`.
- The Linux behaviour of the Glyph app is unverified; macOS was exercised in a real pty.
- Glyph's `Log` widget cannot be cleared, so the clone log is one continuous stream for the
  session, with a header line per run.
