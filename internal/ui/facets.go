// Package ui is git-explorer's Glyph shell: the Org and Repo panes, their facets and
// selection, and the modal screens around them. state*.go hold all behaviour with no
// terminal dependency and are tested directly; view*.go turn that state into Glyph
// components.
package ui

import (
	"sort"
	"strings"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

type sortMode int

const (
	sortName sortMode = iota
	sortActivity
	sortAffiliation // Org pane only
)

func (s sortMode) String() string {
	switch s {
	case sortActivity:
		return "activity"
	case sortAffiliation:
		return "affiliation"
	default:
		return "name"
	}
}

type tri int

const (
	triHide tri = iota
	triShow
	triOnly
)

func (t tri) String() string {
	switch t {
	case triShow:
		return "show"
	case triOnly:
		return "only"
	default:
		return "hide"
	}
}

func (t tri) next() tri { return (t + 1) % 3 }

func (t tri) passes(fieldSet bool) bool {
	switch t {
	case triShow:
		return true
	case triOnly:
		return fieldSet
	default:
		return !fieldSet
	}
}

type visFilter int

const (
	visAll visFilter = iota
	visPublic
	visPrivate
	visInternal
)

func (v visFilter) String() string {
	switch v {
	case visPublic:
		return "public"
	case visPrivate:
		return "private"
	case visInternal:
		return "internal"
	default:
		return "all"
	}
}

func (v visFilter) next() visFilter { return (v + 1) % 4 }

func (v visFilter) passes(got forge.Visibility) bool {
	switch v {
	case visPublic:
		return got == forge.VisibilityPublic
	case visPrivate:
		return got == forge.VisibilityPrivate
	case visInternal:
		return got == forge.VisibilityInternal
	default:
		return true
	}
}

type affFilter int

const (
	affAll affFilter = iota
	affOwner
	affMember
	affCollaborator
	affNone
)

func (a affFilter) String() string {
	switch a {
	case affOwner:
		return "owner"
	case affMember:
		return "member"
	case affCollaborator:
		return "collaborator"
	case affNone:
		return "none"
	default:
		return "any"
	}
}

func (a affFilter) next() affFilter { return (a + 1) % 5 }

func (a affFilter) passes(got forge.Affiliation) bool {
	switch a {
	case affOwner:
		return got == forge.AffiliationOwner
	case affMember:
		return got == forge.AffiliationMember
	case affCollaborator:
		return got == forge.AffiliationCollaborator
	case affNone:
		return got == forge.AffiliationNone
	default:
		return true
	}
}

func lowerLess(a, b string) bool { return strings.ToLower(a) < strings.ToLower(b) }

// sortedOrgs returns orgs passing aff, sorted by name or by Affiliation (Owner first,
// name within each). The input is not modified.
func sortedOrgs(orgs []forge.Org, aff affFilter, mode sortMode) []forge.Org {
	out := make([]forge.Org, 0, len(orgs))
	for _, o := range orgs {
		if aff.passes(o.Affiliation) {
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if mode == sortAffiliation && out[i].Affiliation != out[j].Affiliation {
			return out[i].Affiliation > out[j].Affiliation
		}
		return lowerLess(out[i].Name, out[j].Name)
	})
	return out
}

// sortedRepos returns repos passing the archived/fork/visibility facets, sorted by name
// or by last activity (most recent first). The input is not modified.
func sortedRepos(repos []forge.Repo, archived, fork tri, vis visFilter, mode sortMode) []forge.Repo {
	out := make([]forge.Repo, 0, len(repos))
	for _, r := range repos {
		if archived.passes(r.Archived) && fork.passes(r.Fork) && vis.passes(r.Visibility) {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if mode == sortActivity {
			return out[i].PushedAt.After(out[j].PushedAt)
		}
		return lowerLess(out[i].Name, out[j].Name)
	})
	return out
}
