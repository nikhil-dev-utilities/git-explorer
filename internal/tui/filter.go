package tui

import (
	"regexp"
	"strings"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// filterOrgs narrows orgs by query: substring match, case-insensitive, unless query
// starts with "/", which switches to regex matching against the remainder — a Repo or
// Org name can never begin with "/", so this is unambiguous (DESIGN.md).
func filterOrgs(orgs []forge.Org, query string) []forge.Org {
	if query == "" {
		return orgs
	}

	if strings.HasPrefix(query, "/") {
		pattern := query[1:]
		re, err := regexp.Compile(pattern)
		if err != nil {
			// An incomplete/invalid regex (e.g. still being typed) matches
			// nothing rather than erroring the whole pane.
			return nil
		}
		out := make([]forge.Org, 0, len(orgs))
		for _, o := range orgs {
			if re.MatchString(o.Name) {
				out = append(out, o)
			}
		}
		return out
	}

	lower := strings.ToLower(query)
	out := make([]forge.Org, 0, len(orgs))
	for _, o := range orgs {
		if strings.Contains(strings.ToLower(o.Name), lower) {
			out = append(out, o)
		}
	}
	return out
}
