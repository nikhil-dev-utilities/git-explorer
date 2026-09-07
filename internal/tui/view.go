package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	switch m.mode {
	case ModeBrowse:
		return m.viewBrowse()
	case ModeLeavePrompt:
		return m.viewLeavePrompt()
	case ModeCloneDialog:
		return m.viewCloneDialog()
	case ModeHostSwitch:
		return m.viewHostSwitch()
	case ModeFatal:
		return m.viewFatal()
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
	return b.String()
}

// viewCloneDialog shows the exact destination path for every selected Repo — sourced
// from clonePreviewResults, which clone.TargetPath (via the injected
// ClonePreviewFunc) computed, never a reimplementation of that logic here — plus its
// pre-flight classification, updating live as the org-subdirectory toggle flips.
func (m Model) viewCloneDialog() string {
	var b strings.Builder
	fmt.Fprintf(&b, "clone %d repos to %s\n", m.selectionCount(), m.cloneTarget)
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
	b.WriteString("\n[esc] cancel\n")
	return b.String()
}

func (m Model) viewLeavePrompt() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d repos selected in %s\n\n", m.selectionCount(), m.currentOrg.Name)
	b.WriteString("[c] clone now  [d] discard  [esc] stay\n")
	return b.String()
}

// viewBrowse renders the Org and Repo panes side by side, Finder-style — both
// visible at once, with focus determining which one receives key input, not which
// one is shown. Below tooNarrowWidth the two-column layout is abandoned for a single
// message rather than rendering something broken or overlapping.
//
// A width of 0 means no tea.WindowSizeMsg has arrived yet (only possible in tests
// that call View() directly without going through a real Bubble Tea Program, which
// always sends one immediately at startup) — treated as "wide enough," so those
// tests see the same pane content this package's earlier slices always rendered.
func (m Model) viewBrowse() string {
	if m.width == 0 {
		// No real terminal size known — render both panes at their natural width,
		// full detail, no column constraint. Only reachable when View() is called
		// directly without going through a real Bubble Tea Program.
		pane := lipgloss.JoinHorizontal(lipgloss.Top, m.viewOrgPane(), " ", m.viewRepoPane(repoDetailFull))
		if status := m.statusLine(); status != "" {
			return pane + "\n" + status
		}
		return pane
	}

	if m.width < tooNarrowWidth {
		return "terminal too narrow\n"
	}

	repoWidth := m.width - orgPaneWidth - 1
	orgCol := lipgloss.NewStyle().Width(orgPaneWidth).Render(m.viewOrgPane())
	repoCol := lipgloss.NewStyle().Width(repoWidth).Render(m.viewRepoPane(detailForWidth(repoWidth)))

	pane := lipgloss.JoinHorizontal(lipgloss.Top, orgCol, " ", repoCol)
	if status := m.statusLine(); status != "" {
		return pane + "\n" + status
	}
	return pane
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
