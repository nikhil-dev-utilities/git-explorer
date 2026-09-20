# The filter is modal, and Repos load on demand with a session cache

Two related changes to how the Org and Repo panes behave.

## Modal filter

`/` starts typing into the focused pane's filter; Enter keeps it and returns to the list,
Esc clears it. Outside that mode every key is a verb. This supersedes ADR-0006, which chose
an always-live filter and rejected exactly this design, and the keymap part of ADR-0008.

The always-live filter made every letter text, which is why the keymap was full of ctrl
chords and why Tab could not both tick a Repo and switch panes. Going modal costs one
keystroke per filter (`/tf` Enter, then `a`, versus `tf` then `^a`) and buys bare keys:
`Space` ticks, `a` toggles tick-all, `x` clears the Selection, `r` reloads, `o` opens
options, `?` help, `Tab` / `Shift-Tab` switch panes. fzf's space-as-AND stays usable because
space is only a tick key outside the filter.

Stray typing now acts, so the destructive-looking verbs are cheap by design: Clone opens a
confirm screen, `a` and `x` only change ticks, and quit stays `^c` (a bare `q` would quit
mid-word). The forbidden-key rule from ADR-0006 still applies, but only inside filter mode.

## Repos load on demand

Focusing a pane never loads anything. Orgs load at startup, on host switch and on reload.
Repos load the first time an Org is opened in a session, or on `r` / `F5`. Reopening the Org
already shown only moves focus and keeps its filter and ticks (this used to refetch and wipe
both every time).

Repos are also cached per Host and Org **in memory, for the session**, so reopening another
Org you already viewed is instant. ADR-0007 rejected an on-disk cache and said any cache must
come with an explicit refresh affordance and a visible staleness indicator; this has both:
`r` / `F5` / the options menu refetch, and the Repo title shows `loaded 4m ago`. Nothing is
written to disk and nothing survives a restart.

## The selection guard moves to the point of loss

ADR-0005 prompted (clone now / discard / stay) when leaving the Repo pane with ticks. With
panes switched by Tab, leaving a pane loses nothing, so the prompt now appears only when ticks
would actually be discarded: opening a *different* Org, or switching Host. Tab, Esc and `h`
keep the ticks.

## Consequences

- Within a session the Repo list can be stale until reloaded; the title says by how much.
- Ticks for Repos that disappeared on reload are dropped.
- The keymap is documented in DESIGN.md and README.md; `internal/ui/keymap_test.go` keeps the
  README table equal to the app's own table.
