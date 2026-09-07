# Selections are Org-scoped and Clone Runs are modal

A Selection belongs to exactly one Org and never spans Orgs, and a Clone Run takes over
the screen until it finishes. Both choices trade capability for simpler state, and they
are recorded together because their costs compound.

For a tool whose stated purpose is bulk cloning, refusing a cross-Org selection is the
surprising part. The rejected alternative was a persistent cart: tick repos in `acme`,
move to `globex`, tick more, clone all of them in one run. It is more capable, and it was
turned down because it creates ticked-but-off-screen state — a selection you cannot see
is a selection you lose track of, and the failure mode is cloning the wrong set into a
directory you then have to unpick by hand. Leaving an Org with a non-empty Selection
instead prompts: clone now, discard, or stay.

Modal Clone Runs were chosen over backgrounded ones for the same reason: one run at a
time needs no run registry, no concurrent-run state, and no handling for quitting with
work in flight.

## Consequences

- Cloning across four Orgs is four sequential, blocking runs. "Clone now" on the way out
  of an Org means waiting for it before reaching the next Org.
- **If the tool ever feels tedious in real use, this pair is why.** The two decisions are
  separable: backgrounding Clone Runs while keeping Selections Org-scoped recovers most
  of the fluency and reintroduces none of the invisible-selection problem. That is the
  cheaper half to reverse, and the one to reverse first.
