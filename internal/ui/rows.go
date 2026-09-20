package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// orgRow and repoRow are what the FilterLists hold. Glyph reads their string and bool
// fields by pointer every frame, so everything a row shows is precomputed into a field.
type orgRow struct {
	Org forge.Org
	Aff string
}

type repoRow struct {
	Repo   forge.Repo
	Ticked bool
	Badges string
	Age    string
}

func newOrgRow(o forge.Org) orgRow {
	return orgRow{Org: o, Aff: o.Affiliation.String()}
}

func newRepoRow(r forge.Repo, ticked bool, now time.Time) repoRow {
	var badges []string
	if r.Archived {
		badges = append(badges, "archived")
	}
	if r.Fork {
		badges = append(badges, "fork")
	}
	if r.Visibility != forge.VisibilityPublic {
		badges = append(badges, r.Visibility.String())
	}
	return repoRow{Repo: r, Ticked: ticked, Badges: strings.Join(badges, " "), Age: ago(now, r.PushedAt)}
}

// ago renders a short relative age such as "3h ago"; zero time renders empty.
func ago(now, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/24/365))
	}
}
