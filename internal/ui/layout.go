package ui

// The Org pane's width cycles through orgPaneWidths; the Repo pane takes the rest and
// sheds columns as it shrinks (age first, then badges), keeping the name longest.
// Below tooNarrowWidth (whole terminal) the two-pane layout gives way to one message.
const (
	tooNarrowWidth = 60

	repoPaneFullWidth  = 55 // >= this: name + badges + age
	repoPaneNoAgeWidth = 46 // >= this: name + badges
	minRepoPaneWidth   = 20
	paneBorderCols     = 2
	defaultOrgWidthIdx = 1
)

var orgPaneWidths = []int{28, 34, 42, 52}

type layout struct {
	tooNarrow  bool
	orgWidth   int16
	showBadges bool
	showAge    bool
}

// layoutFor computes the layout for a terminal width and Org width preset. A wide
// preset on a narrow terminal shrinks so the Repo pane keeps minRepoPaneWidth.
func layoutFor(width, orgIdx int) layout {
	if width < tooNarrowWidth {
		return layout{tooNarrow: true, orgWidth: int16(orgPaneWidths[orgIdx])}
	}
	orgW := orgPaneWidths[orgIdx]
	if max := width - paneBorderCols*2 - minRepoPaneWidth; orgW > max {
		orgW = max
	}
	repoW := width - orgW - paneBorderCols*2
	return layout{
		orgWidth:   int16(orgW),
		showBadges: repoW >= repoPaneNoAgeWidth,
		showAge:    repoW >= repoPaneFullWidth,
	}
}
