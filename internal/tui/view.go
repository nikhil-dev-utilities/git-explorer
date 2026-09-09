package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

func (m Model) View() string {
	switch m.mode {
	case ModeBrowse:
		return m.viewBrowse()
	case ModeLeavePrompt:
		return m.viewLeavePrompt()
	case ModeCloneDialog:
		return m.viewCloneDialog()
	case ModeCloneRun:
		return m.viewCloneRun()
	case ModeHostSwitch:
		return m.viewHostSwitch()
	case ModeFatal:
		return m.viewFatal()
	case ModeHelp:
		return m.viewHelp()
	}
	return ""
}

// viewFatal is the whole screen: nothing else in the app works until this is fixed,
// so nothing else is shown. The message already names the fix command — Forge
// supplies that text (PRD 1) — this package never re-derives or duplicates it.
func (m Model) viewFatal() string {
	var b strings.Builder
	b.WriteString("Fatal\n\n")
	if m.fatalErr != nil {
		fmt.Fprintf(&b, "%v\n\n", m.fatalErr)
	}
	b.WriteString(renderButtons(fatalButtons()))
	b.WriteString("\n")
	return b.String()
}

// fatalButtons has no primary: neither action is a "safer default" the way a
// confirm dialog's cancel/stay option is — switching Host and quitting are just
// two different escapes, not a safe-vs-unsafe choice.
func fatalButtons() []button {
	return []button{
		{key: "^y", label: "switch host"},
		{key: "^c", label: "quit"},
	}
}

// viewHelp lists every binding straight from keymapTable — the same table the
// regression test in keymap_test.go checks — grouped by mode, so this listing can
// never show something the test didn't also verify.
func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString("help\n\n")

	currentSection := ""
	for _, kb := range keymapTable {
		if kb.mode != currentSection {
			currentSection = kb.mode
			fmt.Fprintf(&b, "%s\n", currentSection)
		}
		fmt.Fprintf(&b, "  %-14s %-8s %s\n", kb.key, kb.altKey, kb.action)
	}

	b.WriteString("\n[esc] close\n")
	return b.String()
}

// statusLine renders a Transient error (a rate limit, typically), with its
// RetryAfter, or nothing when there isn't one. It never replaces or obscures pane
// content — callers append it, they don't return it in place of the pane.
func (m Model) statusLine() string {
	if m.transientErr == nil {
		return ""
	}
	if d := retryAfter(m.transientErr); d > 0 {
		return fmt.Sprintf("%v — retrying in %s\n", m.transientErr, d.Round(time.Second))
	}
	return fmt.Sprintf("%v\n", m.transientErr)
}

func (m Model) viewHostSwitch() string {
	var b strings.Builder
	b.WriteString("hosts\n\n")
	for i, h := range m.hosts {
		cursor := "  "
		if i == m.hostCursor {
			cursor = "> "
		}
		active := " "
		if i == m.activeHostIdx {
			active = "*"
		}
		fmt.Fprintf(&b, "%s%s %s\n", cursor, active, h.Name)
	}
	b.WriteString("\n[↑/↓] move\n")
	b.WriteString(renderButtons(hostSwitchButtons()))
	b.WriteString("\n")
	return b.String()
}

// hostSwitchButtons: cancel is primary — leaving everything unchanged is the safe
// default when you've opened the switcher but haven't committed to a choice.
func hostSwitchButtons() []button {
	return []button{
		{key: "enter", label: "switch"},
		{key: "esc", label: "cancel", primary: true},
		{key: "^c", label: "quit"},
	}
}

// viewCloneDialog shows the exact destination path for every selected Repo — sourced
// from clonePreviewResults, which clone.TargetPath (via the injected
// ClonePreviewFunc) computed, never a reimplementation of that logic here — plus its
// pre-flight classification, updating live as the org-subdirectory toggle flips or
// the target path itself is edited (see editCloneTarget).
func (m Model) viewCloneDialog() string {
	var b strings.Builder
	fmt.Fprintf(&b, "clone %d repos\n", m.selectionCount())
	fmt.Fprintf(&b, "target: %s (type to edit)\n", m.cloneTarget)
	subdir := "off"
	if m.cloneOrgSubdir {
		subdir = "on"
	}
	fmt.Fprintf(&b, "[tab] org-subdirectory: %s\n\n", subdir)

	if len(m.clonePreviewResults) == 0 {
		b.WriteString("classifying...\n")
		return b.String()
	}

	for _, r := range m.clonePreviewResults {
		fmt.Fprintf(&b, "%-30s %-10s %s\n", r.Repo.Name, r.Outcome, r.Dest)
	}
	b.WriteString("\n")
	b.WriteString(renderButtons(cloneDialogButtons()))
	b.WriteString("\n")
	return b.String()
}

// cloneDialogButtons: cancel is primary — cloning is the one-way action here (a
// Clone Run can conflict-skip its way around existing paths, but it still writes
// to disk), cancel is the reversible default.
func cloneDialogButtons() []button {
	return []button{
		{key: "enter", label: "clone"},
		{key: "esc", label: "cancel", primary: true},
	}
}

// viewCloneRun shows an indeterminate in-flight state while the single tea.Cmd
// wrapping CloneRunnerFunc is running (see clonerun.go's doc comment for why this
// isn't per-Repo live progress), then the full breakdown — grouped by Outcome, with
// failures called out individually — once cloneRunResultMsg arrives.
func (m Model) viewCloneRun() string {
	var b strings.Builder

	if m.cloneRunInFlight {
		b.WriteString("cloning...\n\n[^c] cancel\n")
		return b.String()
	}

	var cloned, skipped, conflict, failed []clone.Result
	for _, r := range m.cloneRunResults {
		switch {
		case r.Err != nil:
			failed = append(failed, r)
		case r.Outcome == clone.OutcomeSkipped:
			skipped = append(skipped, r)
		case r.Outcome == clone.OutcomeConflict:
			conflict = append(conflict, r)
		default:
			cloned = append(cloned, r)
		}
	}

	fmt.Fprintf(&b, "cloned: %d  skipped: %d  conflict: %d  failed: %d\n\n",
		len(cloned), len(skipped), len(conflict), len(failed))

	if len(failed) > 0 {
		b.WriteString("failed:\n")
		for _, r := range failed {
			fmt.Fprintf(&b, "  %-30s %v\n", r.Repo.Name, r.Err)
		}
		b.WriteString("\n[r] retry failures")
	}
	b.WriteString("  [esc] done\n")
	return b.String()
}

func (m Model) viewLeavePrompt() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d repos selected in %s\n\n", m.selectionCount(), m.currentOrg.Name)
	b.WriteString(renderButtons(leavePromptButtons()))
	b.WriteString("\n")
	return b.String()
}

// leavePromptButtons: stay is primary — of the three, it's the only one that
// commits to nothing, the safest response to a prompt you weren't necessarily
// expecting.
func leavePromptButtons() []button {
	return []button{
		{key: "c", label: "clone now"},
		{key: "d", label: "discard"},
		{key: "esc", label: "stay", primary: true},
	}
}

// paneBorderCols is how many columns a bordered pane consumes beyond its content
// width (one column each side). paneGapCols is the blank column left between the
// two bordered panes. paneBorderRows is the same for height (one row each side).
const (
	paneBorderCols = 2
	paneGapCols    = 1
	paneBorderRows = 2
)

// viewBrowse renders the Org and Repo panes side by side, Finder-style — both
// visible at once, with focus determining which one receives key input, not which
// one is shown. Each pane gets its own bounding box, stretched to fill the terminal's
// full height (short of what the footer needs below it) — the focused pane's border
// is a distinct color, so focus is legible at a glance. Below tooNarrowWidth the
// two-column layout is abandoned for a single message rather than rendering something
// broken or overlapping.
//
// A width of 0 means no tea.WindowSizeMsg has arrived yet (only possible in tests
// that call View() directly without going through a real Bubble Tea Program, which
// always sends one immediately at startup) — treated as "wide enough," so those
// tests see the same pane content this package's earlier slices always rendered, with
// no border or height constraint (there being no real terminal height to fill).
func (m Model) viewBrowse() string {
	if m.width == 0 {
		pane := lipgloss.JoinHorizontal(lipgloss.Top, m.viewOrgPane(0), " ", m.viewRepoPane(repoDetailFull, 0))
		return m.withBrowseFooter(pane, browseFooterFallbackWidth)
	}

	if m.width < tooNarrowWidth {
		return "terminal too narrow\n"
	}

	repoWidth := m.width - orgPaneWidth - paneGapCols - paneBorderCols*2
	footerRows := m.browseFooterRows(m.width)
	contentHeight := m.paneContentHeight(footerRows)

	orgColor, repoColor := m.paneBorderColors()
	orgCol := m.paneStyle(orgPaneWidth, orgColor, footerRows).Render(m.viewOrgPane(contentHeight))
	repoCol := m.paneStyle(repoWidth, repoColor, footerRows).Render(m.viewRepoPane(detailForWidth(repoWidth), contentHeight))

	pane := lipgloss.JoinHorizontal(lipgloss.Top, orgCol, " ", repoCol)
	return m.withBrowseFooter(pane, m.width)
}

// paneBorderColors picks each pane's border color from focus — exactly one of the
// two is ever focusedBorderColor, so which pane has input focus is legible from the
// border alone, without reading any text.
func (m Model) paneBorderColors() (orgColor, repoColor lipgloss.Color) {
	orgColor, repoColor = blurredBorderColor, blurredBorderColor
	if m.focus == FocusOrgs {
		orgColor = focusedBorderColor
	} else {
		repoColor = focusedBorderColor
	}
	return orgColor, repoColor
}

// paneStyle is the shared bordered-box style for a Browse pane: the given content
// width, bordered and colored by focus, stretched to fill paneContentHeight(footerRows)
// — only called once m.height is known (the m.width == 0 branch above never reaches
// this).
func (m Model) paneStyle(width int, color lipgloss.Color, footerRows int) lipgloss.Style {
	style := lipgloss.NewStyle().Width(width).Border(lipgloss.RoundedBorder()).BorderForeground(color)
	return style.Height(m.paneContentHeight(footerRows))
}

// paneContentHeight is how many rows a Browse pane's box has for its own content —
// header lines and the (possibly windowed) list together — short of footerRows, the
// rows the footer below it will take (see browseFooterRows). Shared by paneStyle
// (which constrains the box to this height) and viewBrowse (which passes it to
// viewOrgPane/viewRepoPane so they know their own list-windowing budget) — the two
// must agree, or the box's fixed height and what's actually rendered inside it drift
// apart, which is exactly the bug class visibleWindow's own doc comment describes.
func (m Model) paneContentHeight(footerRows int) int {
	contentHeight := m.height - paneBorderRows - footerRows
	if contentHeight < 1 {
		contentHeight = 1
	}
	return contentHeight
}

// browseFooterFallbackWidth is used only when m.width == 0 (View() called directly,
// no real terminal) — wide enough for the key-hint grid to lay out at a reasonable
// column count without a real width to measure against.
const browseFooterFallbackWidth = 80

// browseFooterRows is how many screen rows withBrowseFooter(_, width) will occupy:
// the compact status line, the key-hint grid (however many rows it wraps to at this
// width), and the transient status line when present. Computed separately from
// withBrowseFooter itself so viewBrowse can size the panes above it before
// rendering the footer text.
func (m Model) browseFooterRows(width int) int {
	rows := 1 + keyHintGridRows(m.currentBrowseKeyHints(), width) // status line + hint grid
	if m.statusLine() != "" {
		rows++
	}
	return rows
}

// withBrowseFooter appends the persistent footer below the panes: a compact status
// line (active Host, live Selection count), then a nano/mc-style key-hint grid
// listing the actions someone actually reaches for (F1 still owns the exhaustive
// listing), then the transient status line, if any. This is the one place in Browse
// mode a user gets any on-screen hint of what to press, so it's never conditional
// on anything: it's there on the very first frame and every frame after.
//
// Both renderKeyHintGrid and statusLine terminate their own last line with "\n" (by
// design — see renderKeyHintGrid's doc comment), so the assembled string ends with
// exactly one trailing newline. That has to come off: the panes above are already
// sized to consume every row browseFooterRows accounted for, and a real terminal's
// alt-screen buffer has no scrollback to absorb an extra line — it scrolls its own
// fixed viewport instead, which pushes the *top* row (the pane borders' top edge)
// out of view. Confirmed against a real terminal, not just reasoned about: this was
// a genuine bug, not a defensive trim for a theoretical case.
func (m Model) withBrowseFooter(pane string, width int) string {
	var b strings.Builder
	b.WriteString(pane)
	b.WriteString("\n")
	b.WriteString(m.browseStatusLine())
	b.WriteString("\n")
	b.WriteString(renderKeyHintGrid(m.currentBrowseKeyHints(), width))
	b.WriteString(m.statusLine())
	return strings.TrimSuffix(b.String(), "\n")
}

// browseStatusLine is deliberately just the two things that change from moment to
// moment — active Host and live Selection count — never key hints, which live in
// the grid below it.
func (m Model) browseStatusLine() string {
	return fmt.Sprintf(" host: %s · %d selected", m.activeHost().Name, m.selectionCount())
}

// viewOrgPane renders the Org list windowed to keep m.orgCursor always visible —
// see visibleWindow's own doc comment for why this matters once there are more
// Orgs than fit in the pane's height. listHeightBudget is the pane box's total
// content-row budget (header lines plus the list together); 0 means unbounded,
// used only by the m.width == 0 test-only View() path.
// renderFilterLine labels the always-live quick-filter row so it's never a bare,
// unlabeled blank line — indistinguishable from a rendering glitch — when no filter
// is typed yet. See issue #87: reported live as "the top pane line gets truncated,"
// which was actually this line carrying no visible affordance at all when empty.
func renderFilterLine(query string) string {
	if query == "" {
		return "search: (type to filter)"
	}
	return "search: " + query
}

func (m Model) viewOrgPane(listHeightBudget int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", renderFilterLine(m.orgFilter))
	if m.orgAffiliation != AffiliationFilterAll {
		fmt.Fprintf(&b, "affiliation: %s\n", affiliationFilterLabel(m.orgAffiliation))
	}

	if m.orgsErr != nil {
		fmt.Fprintf(&b, "error loading orgs: %v  [^r] retry\n", m.orgsErr)
		if len(m.orgs) == 0 {
			return b.String()
		}
		b.WriteString("(showing what already loaded)\n")
	}

	visible := filterOrgsByAffiliation(m.orgs, m.orgFilter, m.orgAffiliation)

	if len(visible) == 0 {
		switch {
		case !m.orgsLoaded && len(m.orgs) == 0:
			b.WriteString("loading...\n")
		case m.orgFilter != "" && len(m.orgs) > 0:
			fmt.Fprintf(&b, "no matches for %q\n", m.orgFilter)
		default:
			b.WriteString("no orgs\n")
		}
		return b.String()
	}

	maxRows := 0 // unbounded — see listHeightBudget's doc comment above
	if listHeightBudget > 0 {
		maxRows = listHeightBudget - strings.Count(b.String(), "\n")
		if maxRows < 1 {
			maxRows = 1
		}
	}
	window, offset := visibleWindow(visible, m.orgCursor, maxRows)

	for i, o := range window {
		cursor := "  "
		if offset+i == m.orgCursor {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%-20s %s\n", cursor, o.Name, o.Affiliation)
	}
	return b.String()
}

// viewRepoPane renders the Repo list windowed to keep m.repoCursor always visible —
// see viewOrgPane's doc comment; same reasoning, same visibleWindow helper.
// listHeightBudget is the pane box's total content-row budget; 0 means unbounded,
// used only by the m.width == 0 test-only View() path.
func (m Model) viewRepoPane(detail repoDetail, listHeightBudget int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "repos: %s · %d selected\n", m.currentOrg.Name, m.selectionCount())
	fmt.Fprintf(&b, "%s\n", renderFilterLine(m.repoFilter))
	fmt.Fprintf(&b, "archived: %s · fork: %s · visibility: %s · sort: %s\n",
		triStateLabel(m.archivedFilter), triStateLabel(m.forkFilter),
		visibilityFilterLabel(m.visibility), sortModeLabel(m.repoSort))

	if m.reposErr != nil {
		fmt.Fprintf(&b, "error loading repos: %v  [^r] retry\n", m.reposErr)
		return b.String()
	}

	visible := sortRepos(filterRepos(m.repos, m.repoFilter, m.archivedFilter, m.forkFilter, m.visibility), m.repoSort)

	if len(visible) == 0 {
		switch {
		case !m.reposLoaded:
			b.WriteString("loading...\n")
		case m.repoFilter != "" && len(m.repos) > 0:
			fmt.Fprintf(&b, "no matches for %q\n", m.repoFilter)
		default:
			b.WriteString("no repos\n")
		}
		return b.String()
	}

	maxRows := 0 // unbounded — see listHeightBudget's doc comment above
	if listHeightBudget > 0 {
		maxRows = listHeightBudget - strings.Count(b.String(), "\n")
		if maxRows < 1 {
			maxRows = 1
		}
	}
	window, offset := visibleWindow(visible, m.repoCursor, maxRows)

	for i, r := range window {
		cursor := "  "
		if offset+i == m.repoCursor {
			cursor = "> "
		}
		tick := "[ ]"
		if m.selected[r.Name] {
			tick = "[x]"
		}

		if detail == repoDetailNameOnly {
			fmt.Fprintf(&b, "%s%s %s\n", cursor, tick, r.Name)
			continue
		}

		badges := ""
		if r.Archived {
			badges += "archived "
		}
		if r.Fork {
			badges += "fork "
		}
		if detail == repoDetailNoDate {
			fmt.Fprintf(&b, "%s%s %-30s %s\n", cursor, tick, r.Name, badges)
			continue
		}
		fmt.Fprintf(&b, "%s%s %-30s %s%s\n", cursor, tick, r.Name, badges, r.PushedAt.Format("2006-01-02"))
	}
	return b.String()
}

func affiliationFilterLabel(a AffiliationFilter) string {
	switch a {
	case AffiliationFilterOwner:
		return "owner"
	case AffiliationFilterMember:
		return "member"
	case AffiliationFilterCollaborator:
		return "collaborator"
	case AffiliationFilterNone:
		return "none"
	default:
		return "any"
	}
}

func triStateLabel(t TriState) string {
	switch t {
	case TriShow:
		return "show"
	case TriOnly:
		return "only"
	default:
		return "hide"
	}
}

func visibilityFilterLabel(v VisibilityFilter) string {
	switch v {
	case VisibilityFilterPublic:
		return "public"
	case VisibilityFilterPrivate:
		return "private"
	case VisibilityFilterInternal:
		return "internal"
	default:
		return "all"
	}
}

func sortModeLabel(s SortMode) string {
	if s == SortByActivity {
		return "activity"
	}
	return "name"
}
