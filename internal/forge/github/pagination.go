package github

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// defaultPerPage is the page size used by fetchAllPages. A page returning fewer than
// this many entries is taken to be the last page — the same heuristic
// streamOrgsPrivate's own pagination already uses.
const defaultPerPage = 100

// fetchAllPages calls endpoint repeatedly with an incrementing page argument,
// decoding and concatenating every page's JSON array body into one slice, until a
// page comes back with fewer than defaultPerPage entries. It exists because
// fetchMemberships, fetchCollaboratorOrgs, and ListRepos each used to fetch a single
// page and silently drop everything past the first 100 results — invisible on a
// small account, but a real gap for an Org or Host with more than that.
//
// Unlike streamOrgsPrivate's own pagination, callers here want the whole list before
// doing anything with it (deduplicating Org names, sorting, etc.), so this returns a
// complete slice rather than streaming pages — there is no equivalent caller-visible
// benefit to streaming for these three.
func fetchAllPages[T any](ctx context.Context, a *Adapter, host forge.Host, endpoint string, extraArgs ...string) ([]T, error) {
	var all []T
	for page := 1; ; page++ {
		args := append(append([]string{}, extraArgs...),
			"-f", fmt.Sprintf("page=%d", page),
			"-f", fmt.Sprintf("per_page=%d", defaultPerPage),
		)
		stdout, err := a.runAPI(ctx, host, endpoint, args...)
		if err != nil {
			return nil, err
		}

		var entries []T
		if err := json.Unmarshal(stdout, &entries); err != nil {
			return nil, unmarshalError(endpoint, err)
		}
		all = append(all, entries...)

		if len(entries) < defaultPerPage {
			return all, nil
		}
	}
}
