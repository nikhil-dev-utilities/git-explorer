package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// orgsPerPage is the page size used when paginating the instance-wide organizations
// endpoint. A page returning fewer than this many entries is taken to be the last page.
const orgsPerPage = 100

// streamOrgsPrivate implements Org discovery for a Private Host (ADR-0002): a
// genuinely different path from the Public Host case, not a parameter change on the
// same one. It fetches the two small, bounded Affiliation-determining calls once
// (membership and the collaborator probe — the same ones the Public Host path uses),
// then paginates the instance-wide organizations endpoint, sending one OrgPage per page
// as it arrives. This is the endpoint that can be large on a real Private Host, so
// streaming it — rather than waiting for a full enumeration — is the point: a caller
// can start navigating after roughly one request's worth of latency.
//
// Every failure, including on the two prerequisite calls, is delivered as the final
// OrgPage's Err rather than a synchronous return, so the caller has one uniform place
// to observe a failure regardless of when it happened. A failure after earlier pages
// already sent does not retract them — the stream simply ends.
func (a *Adapter) streamOrgsPrivate(ctx context.Context, host forge.Host, out chan<- forge.OrgPage) {
	defer close(out)

	memberships, err := a.fetchMemberships(ctx, host)
	if err != nil {
		out <- forge.OrgPage{Err: err}
		return
	}
	collaboratorOrgs, err := a.fetchCollaboratorOrgs(ctx, host)
	if err != nil {
		out <- forge.OrgPage{Err: err}
		return
	}
	affiliationByName := buildAffiliationIndex(memberships, collaboratorOrgs)

	var since int64
	for batch := 1; ; batch++ {
		entries, more, err := a.fetchOrgsPage(ctx, host, since)
		if err != nil {
			out <- forge.OrgPage{Err: err}
			return
		}

		orgs := make([]forge.Org, len(entries))
		for i, e := range entries {
			// A name with no entry gets AffiliationNone, the zero value — a normal,
			// expected, displayed result on a Private Host, not an omission.
			orgs[i] = forge.Org{Name: e.Login, Host: host, Affiliation: affiliationByName[e.Login]}
		}
		slog.InfoContext(ctx, "fetched orgs page", "host", host.Name, "batch", batch, "since", since, "count", len(orgs), "more", more)
		out <- forge.OrgPage{Orgs: orgs}

		if !more {
			return
		}
		// GET /organizations paginates exclusively via since (an org ID cursor), not
		// page — see fetchOrgsPage's doc comment. entries is ID-ordered ascending, so
		// the last entry's ID is the correct next cursor.
		since = entries[len(entries)-1].ID
	}
}

func buildAffiliationIndex(memberships []forge.Org, collaboratorOrgs []string) map[string]forge.Affiliation {
	idx := make(map[string]forge.Affiliation, len(memberships)+len(collaboratorOrgs))
	for _, org := range memberships {
		idx[org.Name] = org.Affiliation
	}
	for _, name := range collaboratorOrgs {
		if _, exists := idx[name]; !exists {
			idx[name] = forge.AffiliationCollaborator
		}
	}
	return idx
}

type instanceOrgEntry struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}

// fetchOrgsPage fetches one page of the instance-wide organizations listing. This
// call is deliberately never made on a Public Host — see listOrgsPublic's doc
// comment for why it's unusable there.
//
// GET /organizations does not support GitHub's usual page= parameter — per GitHub's
// own docs, "Pagination is powered exclusively by the since parameter" (an org ID
// cursor: only organizations with an ID greater than since are returned). Passing
// page= instead is silently ignored, not rejected — every request just returns the
// same first page over and over, forever, which is exactly what happened live: an
// unbounded "fetched orgs page" loop that reached page 1200, with the Org pane
// showing the same handful of names on repeat. since=0 (the caller's first call)
// returns organizations from the beginning.
func (a *Adapter) fetchOrgsPage(ctx context.Context, host forge.Host, since int64) (entries []instanceOrgEntry, more bool, err error) {
	stdout, err := a.runAPI(ctx, host, "organizations",
		"-f", fmt.Sprintf("since=%d", since),
		"-f", fmt.Sprintf("per_page=%d", orgsPerPage),
	)
	if err != nil {
		return nil, false, err
	}

	if err := json.Unmarshal(stdout, &entries); err != nil {
		return nil, false, unmarshalError("organizations", err)
	}
	return entries, len(entries) == orgsPerPage, nil
}
