# Frontdoors are private to a Forge, not a second plugin axis

The tool must support more than one Frontdoor for GitHub (`gh` CLI first, then
REST/GraphQL), and may one day support another Forge. We deliberately did **not** make
these two composable plugin axes. `Forge` is the only port the TUI depends on; each Forge
adapter privately chooses its own Frontdoor, and the Frontdoor interface is unexported.

The rejected alternative — exporting `Transport` and composing `NewGitHub(rest)` — needs
a transport abstraction general enough for two unrelated API shapes, which in practice
degrades into a thin HTTP wrapper that earns nothing. The other rejected alternative, one
flat implementation per (Forge, Frontdoor) pair, duplicates GitHub's pagination,
affiliation and mapping logic across every GitHub variant, and that logic is most of the
code.

## Consequences

- Frontdoor plurality is a GitHub-specific problem. A second Forge may ship with exactly
  one Frontdoor and never grow the concept.
- **GitLab is deferred and the `Forge` port is modelled honestly on GitHub only** — two
  levels, Org → Repo. We are not paying for speculative generality: GitLab's nested
  Groups do not fit a two-level port, and when it is picked up it may well deserve its
  own model and its own implementation rather than being forced through this one. A
  reader should expect the port to change shape at that point, not to already accommodate
  it.
