package tui

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	switch m.mode {
	case ModeBrowse:
		return m.viewBrowse()
	case ModeLeavePrompt:
		return m.viewLeavePrompt()
	case ModeCloneDialog:
		// Built out by a later slice of this PRD (#31).
		return fmt.Sprintf("clone dialog: %d repos selected\n", m.selectionCount())
	case ModeHostSwitch:
		return m.viewHostSwitch()
	}
	return ""
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

func (m Model) viewLeavePrompt() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d repos selected in %s\n\n", m.selectionCount(), m.currentOrg.Name)
	b.WriteString("[c] clone now  [d] discard  [esc] stay\n")
	return b.String()
}

// viewBrowse renders whichever pane has focus. True side-by-side column layout
// (Org pane fixed width, Repo pane taking the remainder, responsive column-shedding)
// is a later slice of this PRD (#30) — this slice's job is the navigation and
// filtering underneath it, not the final two-column composition.
func (m Model) viewBrowse() string {
	switch m.focus {
	case FocusRepos:
		return m.viewRepoPane()
	default:
		return m.viewOrgPane()
	}
}

func (m Model) viewOrgPane() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", m.orgFilter)
	if m.orgAffiliation != AffiliationFilterAll {
		fmt.Fprintf(&b, "affiliation: %s\n", affiliationFilterLabel(m.orgAffiliation))
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

func (m Model) viewRepoPane() string {
	var b strings.Builder

	fmt.Fprintf(&b, "repos: %s · %d selected\n", m.currentOrg.Name, m.selectionCount())
	fmt.Fprintf(&b, "%s\n", m.repoFilter)
	fmt.Fprintf(&b, "archived: %s · fork: %s · visibility: %s · sort: %s\n",
		triStateLabel(m.archivedFilter), triStateLabel(m.forkFilter),
		visibilityFilterLabel(m.visibility), sortModeLabel(m.repoSort))

	if m.reposErr != nil {
		fmt.Fprintf(&b, "error loading repos: %v\n", m.reposErr)
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
		badges := ""
		if r.Archived {
			badges += "archived "
		}
		if r.Fork {
			badges += "fork "
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
