package tui

import (
	"regexp"
	"sort"
	"strings"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// matchesQuery reports whether name passes query: substring match, case-insensitive,
// unless query starts with "/", which switches to regex matching against the
// remainder — a Repo or Org name can never begin with "/", so this is unambiguous
// (DESIGN.md). An invalid/incomplete regex matches nothing rather than erroring the
// whole pane.
func matchesQuery(name, query string) bool {
	if query == "" {
		return true
	}
	if strings.HasPrefix(query, "/") {
		re, err := regexp.Compile(query[1:])
		if err != nil {
			return false
		}
		return re.MatchString(name)
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(query))
}

// filterOrgs narrows orgs by name (via matchesQuery) and by Affiliation.
func filterOrgs(orgs []forge.Org, query string) []forge.Org {
	return filterOrgsByAffiliation(orgs, query, AffiliationFilterAll)
}

func filterOrgsByAffiliation(orgs []forge.Org, query string, aff AffiliationFilter) []forge.Org {
	out := make([]forge.Org, 0, len(orgs))
	for _, o := range orgs {
		if !matchesQuery(o.Name, query) {
			continue
		}
		if !affiliationPasses(o.Affiliation, aff) {
			continue
		}
		out = append(out, o)
	}
	return out
}

func affiliationPasses(got forge.Affiliation, filter AffiliationFilter) bool {
	switch filter {
	case AffiliationFilterOwner:
		return got == forge.AffiliationOwner
	case AffiliationFilterMember:
		return got == forge.AffiliationMember
	case AffiliationFilterCollaborator:
		return got == forge.AffiliationCollaborator
	case AffiliationFilterNone:
		return got == forge.AffiliationNone
	default:
		return true
	}
}

// filterRepos narrows repos by name, then by the archived/fork tri-states and the
// visibility filter, all independent of each other.
func filterRepos(repos []forge.Repo, query string, archived, fork TriState, vis VisibilityFilter) []forge.Repo {
	out := make([]forge.Repo, 0, len(repos))
	for _, r := range repos {
		if !matchesQuery(r.Name, query) {
			continue
		}
		if !triStatePasses(r.Archived, archived) {
			continue
		}
		if !triStatePasses(r.Fork, fork) {
			continue
		}
		if !visibilityPasses(r.Visibility, vis) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func triStatePasses(fieldSet bool, filter TriState) bool {
	switch filter {
	case TriShow:
		return true
	case TriOnly:
		return fieldSet
	default: // TriHide
		return !fieldSet
	}
}

func visibilityPasses(got forge.Visibility, filter VisibilityFilter) bool {
	switch filter {
	case VisibilityFilterPublic:
		return got == forge.VisibilityPublic
	case VisibilityFilterPrivate:
		return got == forge.VisibilityPrivate
	case VisibilityFilterInternal:
		return got == forge.VisibilityInternal
	default:
		return true
	}
}

// sortRepos returns a sorted copy — name (case-insensitive) or last-activity, most
// recent first. The input is never mutated.
func sortRepos(repos []forge.Repo, mode SortMode) []forge.Repo {
	out := make([]forge.Repo, len(repos))
	copy(out, repos)
	switch mode {
	case SortByActivity:
		sort.SliceStable(out, func(i, j int) bool { return out[i].PushedAt.After(out[j].PushedAt) })
	default:
		sort.SliceStable(out, func(i, j int) bool {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		})
	}
	return out
}

// sortOrgs returns a sorted copy — name (case-insensitive) or by Affiliation
// (Owner, then Member, then Collaborator, then None — matching AffiliationFilter's
// own cycle order in model.go), name-tiebroken within each Affiliation. The input is
// never mutated.
func sortOrgs(orgs []forge.Org, mode SortMode) []forge.Org {
	out := make([]forge.Org, len(orgs))
	copy(out, orgs)
	switch mode {
	case SortByAffiliation:
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Affiliation != out[j].Affiliation {
				return out[i].Affiliation > out[j].Affiliation
			}
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		})
	default:
		sort.SliceStable(out, func(i, j int) bool {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		})
	}
	return out
}

func nextTriState(t TriState) TriState {
	return (t + 1) % 3
}

func nextVisibilityFilter(v VisibilityFilter) VisibilityFilter {
	return (v + 1) % 4
}

func nextAffiliationFilter(a AffiliationFilter) AffiliationFilter {
	return (a + 1) % 5
}

func nextSortMode(s SortMode) SortMode {
	if s == SortByName {
		return SortByActivity
	}
	return SortByName
}

// nextOrgSortMode is Org's own two-state cycle — see orgSort's doc comment in
// model.go for why it doesn't share nextSortMode with Repo.
func nextOrgSortMode(s SortMode) SortMode {
	if s == SortByName {
		return SortByAffiliation
	}
	return SortByName
}
