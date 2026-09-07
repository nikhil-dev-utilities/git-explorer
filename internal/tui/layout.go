package tui

// Layout thresholds. The Org pane is a fixed 28 columns; the Repo pane takes the
// remainder and sheds columns as that remainder shrinks — date first, then state
// badges — keeping the Repo name visible longest, since that's the one thing a user
// can't do without. Below tooNarrowWidth (measured on the whole terminal), the
// two-pane layout is abandoned entirely for a single message.
//
// The shedding thresholds are measured against the Repo pane's own width (total
// width minus the Org pane and its divider), not total width — a row needs roughly
// cursor+tick+name+badges+date ≈ 55 columns for full detail, ≈ 45 without the date,
// so a standard 80-column terminal (repo pane ≈ 51 wide) already sheds the date
// column rather than rendering something illegibly cramped.
const (
	orgPaneWidth = 28

	tooNarrowWidth = 60

	repoPaneFullWidth   = 55 // >= this: name + badges + date
	repoPaneNoDateWidth = 45 // >= this: name + badges
	// below repoPaneNoDateWidth: name only
)

// repoDetail is how much per-Repo detail the current width affords.
type repoDetail int

const (
	repoDetailFull     repoDetail = iota // name + badges + date
	repoDetailNoDate                     // name + badges
	repoDetailNameOnly                   // name only
)

// detailForWidth takes the Repo pane's own width (not the terminal's total width).
func detailForWidth(repoWidth int) repoDetail {
	switch {
	case repoWidth >= repoPaneFullWidth:
		return repoDetailFull
	case repoWidth >= repoPaneNoDateWidth:
		return repoDetailNoDate
	default:
		return repoDetailNameOnly
	}
}
