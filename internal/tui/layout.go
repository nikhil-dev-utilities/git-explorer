package tui

import "github.com/charmbracelet/lipgloss"

// Pane border colors: the focused pane's bounding box is visually distinct from the
// blurred one's, so focus is legible at a glance without reading any text. Chosen to
// read reasonably against both light and dark terminal backgrounds.
var (
	focusedBorderColor = lipgloss.Color("62")  // a mid blue/purple
	blurredBorderColor = lipgloss.Color("240") // a neutral gray
)

// Layout thresholds. The Org pane's width cycles through orgPaneWidthPresets (^g /
// alt-w — see cycleOrgPaneWidth); the Repo pane takes the remainder and sheds columns
// as that remainder shrinks — date first, then state badges — keeping the Repo name
// visible longest, since that's the one thing a user can't do without. Below
// tooNarrowWidth (measured on the whole terminal), the two-pane layout is abandoned
// entirely for a single message.
//
// The shedding thresholds are measured against the Repo pane's own content width —
// total terminal width minus the Org pane, both panes' borders, and the gap between
// them (paneBorderCols*2 + the Org pane's current width + paneGapCols; see
// viewBrowse) — not total width. A row needs cursor(2) + tick(3) + " " + name(30) + "
// " + badges(up to "archived " = 9) = 46 columns for name+badges without wrapping (one
// archived-or-fork badge; the rare case of both showing simultaneously can still wrap,
// a pre-existing approximation neither this constant nor repoPaneFullWidth chases),
// plus roughly 10 more for the date.
const (
	tooNarrowWidth = 60

	repoPaneFullWidth   = 55 // >= this: name + badges + date
	repoPaneNoDateWidth = 46 // >= this: name + badges
	// below repoPaneNoDateWidth: name only

	// minRepoPaneWidth is the floor viewBrowse clamps the Org pane's chosen preset
	// against, so a wide preset on a narrow terminal shrinks the Org pane back down
	// rather than crushing the Repo pane to near-nothing (or negative width). It's
	// well under repoPaneNoDateWidth — staying legible, not staying detailed, is the
	// bar down here.
	minRepoPaneWidth = 20
)

// orgPaneWidthPresets is the fixed cycle ^g (alt-w) steps through. Index 0 (28) is
// this feature's pre-existing width, kept as the narrow end of the range rather than
// dropped, since some users on narrower terminals will still want it.
// defaultOrgPaneWidthIdx points at 34 — larger by default, per direct request. Total
// width is fixed, so a wider Org pane necessarily costs the Repo pane's detail level:
// at a stock 80-column terminal even this default already drops the Repo pane to
// name-only (repoPaneNoDateWidth needs orgWidth <= 29 at that width) — a real
// trade-off, not a free win. A 100+ column terminal keeps at least "no date" detail
// at every preset.
var orgPaneWidthPresets = []int{28, 34, 42, 52}

const defaultOrgPaneWidthIdx = 1

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
