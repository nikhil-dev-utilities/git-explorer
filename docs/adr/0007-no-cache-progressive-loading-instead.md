# No cache; progressive loading instead

Nothing is persisted between runs. Org and Repo lists are fetched fresh on every launch
and held in memory only. A Private Host with thousands of Orgs is made usable not by
caching but by streaming pages into the pane as they arrive — the first page is navigable
in roughly the time of one request, and the rest fill in behind a loading indicator.

The obvious alternative is an on-disk cache keyed by Host, which would make every launch
after the first instant. It was rejected because of what this tool does with the data it
shows: it writes to your disk. A stale Repo list means cloning a repo that was renamed,
or missing one created this morning, and the user has no way to tell that is what
happened. Buying a faster second launch with a class of silently-wrong-data bugs is a bad
trade in a tool whose entire output is files on your machine.

Avoiding the cache also removes everything that comes with one: TTL tuning, a manual
refresh key, a cache-clear command, a staleness indicator, and the bug reports that begin
"it's showing a repo I deleted".

## Consequences

- The full fetch is paid on every launch. This is the accepted cost, and progressive
  loading is what makes it tolerable rather than a cache.
- **Someone will propose adding a cache the first time they wait three seconds.** The
  answer is not "no" but "measure it, and if a real Private Host genuinely hurts, add a
  caching decorator around `forge.Forge`" — the port is already shaped for that
  (ADR-0001), so this is reversible without touching the TUI. What must not happen is a
  cache appearing without an explicit refresh affordance and a visible staleness
  indicator, because that is the version that produces wrong clones.
