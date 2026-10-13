# gh-discovered Hosts merge with the hosts: list

> Reverses the rule in DESIGN.md's Config section that an explicit `hosts:` list always
> wins over discovery.

Until now, a config with a `hosts:` key used exactly that list. With no key, git-explorer
used every Host `gh` is logged into. Adding one Bitbucket Host therefore turned discovery
off, and a user lost the GitHub Enterprise Host that `gh` had found for them.

Now discovery always runs, and the `hosts:` list is added to what it finds:

- Discovered Hosts come first. Adding a Host never changes the one that opens at startup.
- A `hosts:` entry with the name of a discovered Host replaces it in place. This is how a
  user sets `protocol` or `default_target` for a discovered Host.
- The other `hosts:` entries follow, in file order.
- `gh_discovery: false` turns discovery off. The Hosts are then exactly the `hosts:` list.

We rejected writing the discovered Hosts into the config file at startup. It would save a
snapshot of `gh`'s login state that goes stale on the next `gh auth login`. It would also
rewrite a file the user owns, and today we only write a starter file when none exists.

## Consequences

- A user who used an explicit list to hide a `gh` Host sees it again. They need
  `gh_discovery: false`.
- Whether a Fatal "not authenticated" error is downgraded to pane-scoped is now decided
  per Host: it is downgraded for any Host not listed in the file. Before, one flag
  covered every Host.
- A config with only Bitbucket Hosts still needs `gh` if `gh` is logged in somewhere,
  because the discovered GitHub Hosts are now present too.
