package github

import (
	"context"
	"encoding/json"
	"fmt"

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

	for page := 1; ; page++ {
		names, more, err := a.fetchOrgsPage(ctx, host, page)
		if err != nil {
			out <- forge.OrgPage{Err: err}
			return
		}

		orgs := make([]forge.Org, len(names))
		for i, name := range names {
			// A name with no entry gets AffiliationNone, the zero value — a normal,
			// expected, displayed result on a Private Host, not an omission.
			orgs[i] = forge.Org{Name: name, Host: host, Affiliation: affiliationByName[name]}
		}
		out <- forge.OrgPage{Orgs: orgs}

		if !more {
			return
		}
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
}

// fetchOrgsPage fetches one page (1-indexed) of the instance-wide organizations
// listing. This call is deliberately never made on a Public Host — see
// listOrgsPublic's doc comment for why it's unusable there.
func (a *Adapter) fetchOrgsPage(ctx context.Context, host forge.Host, page int) (names []string, more bool, err error) {
	stdout, err := a.runAPI(ctx, host, "organizations",
		"-f", fmt.Sprintf("page=%d", page),
		"-f", fmt.Sprintf("per_page=%d", orgsPerPage),
	)
	if err != nil {
		return nil, false, err
	}

	var entries []instanceOrgEntry
	if err := json.Unmarshal(stdout, &entries); err != nil {
		return nil, false, unmarshalError("organizations", err)
	}

	names = make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Login
	}
	return names, len(entries) == orgsPerPage, nil
}
