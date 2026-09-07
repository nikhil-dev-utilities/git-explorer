package tui

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	switch m.mode {
	case ModeBrowse:
		return m.viewBrowse()
	}
	return ""
}

func (m Model) viewBrowse() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", m.orgFilter)

	visible := filterOrgs(m.orgs, m.orgFilter)

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

	for _, o := range visible {
		fmt.Fprintf(&b, "%-20s %s\n", o.Name, o.Affiliation)
	}
	return b.String()
}
