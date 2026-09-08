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
	b.WriteString("[^y] switch host  [^c] quit\n")
	return b.String()
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
	b.WriteString("\n[↑/↓] move  [enter] switch  [esc] cancel  [^c] quit\n")
	return b.String()
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
	b.WriteString("\n[enter] clone  [esc] cancel\n")
	return b.String()
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
	b.WriteString("[c] clone now  [d] discard  [esc] stay\n")
	return b.String()
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
		pane := lipgloss.JoinHorizontal(lipgloss.Top, m.viewOrgPane(), " ", m.viewRepoPane(repoDetailFull))
		return m.withBrowseFooter(pane)
	}

	if m.width < tooNarrowWidth {
		return "terminal too narrow\n"
	}

	repoWidth := m.width - orgPaneWidth - paneGapCols - paneBorderCols*2

	orgColor, repoColor := m.paneBorderColors()
	orgCol := m.paneStyle(orgPaneWidth, orgColor).Render(m.viewOrgPane())
	repoCol := m.paneStyle(repoWidth, repoColor).Render(m.viewRepoPane(detailForWidth(repoWidth)))

	pane := lipgloss.JoinHorizontal(lipgloss.Top, orgCol, " ", repoCol)
	return m.withBrowseFooter(pane)
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
// width, bordered and colored by focus, stretched to fill the terminal's full height
// short of what the footer (and, when present, the status line) needs below it —
// only called once m.height is known (the m.width == 0 branch above never reaches
// this), so this is the only place that height budget is computed.
func (m Model) paneStyle(width int, color lipgloss.Color) lipgloss.Style {
	style := lipgloss.NewStyle().Width(width).Border(lipgloss.RoundedBorder()).BorderForeground(color)

	footerRows := 1
	if m.statusLine() != "" {
		footerRows++
	}
	contentHeight := m.height - paneBorderRows - footerRows
	if contentHeight < 1 {
		contentHeight = 1
	}
	return style.Height(contentHeight)
}

// withBrowseFooter appends the persistent footer DESIGN.md's own mockup shows below
// the panes — active Host, Selection count, and the handful of keys someone actually
// needs in the moment (never the full keymap; F1 already opens that) — followed by
// the transient status line, if any. This is the one place in Browse mode a user gets
// any on-screen hint of what to press, so it's never conditional on anything: it's
// there on the very first frame and every frame after.
func (m Model) withBrowseFooter(pane string) string {
	var b strings.Builder
	b.WriteString(pane)
	b.WriteString("\n")
	b.WriteString(m.browseFooter())
	if status := m.statusLine(); status != "" {
		b.WriteString("\n")
		b.WriteString(status)
	}
	return b.String()
}

func (m Model) browseFooter() string {
	enterHint := "enter descend"
	if m.focus == FocusRepos {
		enterHint = "enter clone"
	}
	return fmt.Sprintf(" host: %s · %d selected · ^y host · %s · F1 help",
		m.activeHost().Name, m.selectionCount(), enterHint)
}

func (m Model) viewOrgPane() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", m.orgFilter)
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

	for i, o := range visible {
		cursor := "  "
		if i == m.orgCursor {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%-20s %s\n", cursor, o.Name, o.Affiliation)
	}
	return b.String()
}

func (m Model) viewRepoPane(detail repoDetail) string {
	var b strings.Builder

	fmt.Fprintf(&b, "repos: %s · %d selected\n", m.currentOrg.Name, m.selectionCount())
	fmt.Fprintf(&b, "%s\n", m.repoFilter)
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

	for i, r := range visible {
		cursor := "  "
		if i == m.repoCursor {
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
