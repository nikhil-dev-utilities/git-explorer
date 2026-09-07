package github

import (
	"context"
	"encoding/json"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// listOrgsPrivate implements Org discovery for a Private Host (ADR-0002): a genuinely
// different path from the Public Host case, not a parameter change on the same one.
// Every Org on the instance is listed via the instance-wide organizations endpoint —
// which is unusable on a Public Host but is exactly the right call on a Private one,
// since seeing Orgs the user doesn't yet belong to is the point. Each is then badged
// with Affiliation computed from the same membership and collaborator-probe calls the
// Public Host path uses. An Org matching neither call keeps AffiliationNone — the zero
// value — rather than being dropped from the result.
func (a *Adapter) listOrgsPrivate(ctx context.Context, host forge.Host) ([]forge.Org, error) {
	all, err := a.fetchAllOrgs(ctx, host)
	if err != nil {
		return nil, err
	}
	memberships, err := a.fetchMemberships(ctx, host)
	if err != nil {
		return nil, err
	}
	collaboratorOrgs, err := a.fetchCollaboratorOrgs(ctx, host)
	if err != nil {
		return nil, err
	}

	affiliationByName := make(map[string]forge.Affiliation, len(memberships)+len(collaboratorOrgs))
	for _, org := range memberships {
		affiliationByName[org.Name] = org.Affiliation
	}
	for _, name := range collaboratorOrgs {
		if _, exists := affiliationByName[name]; !exists {
			affiliationByName[name] = forge.AffiliationCollaborator
		}
	}

	orgs := make([]forge.Org, len(all))
	for i, name := range all {
		// Orgs with no entry get AffiliationNone, the zero value — a normal,
		// expected, displayed result on a Private Host, not an omission.
		orgs[i] = forge.Org{Name: name, Affiliation: affiliationByName[name]}
	}
	return orgs, nil
}

type instanceOrgEntry struct {
	Login string `json:"login"`
}

// fetchAllOrgs lists every Org on the instance via the instance-wide organizations
// endpoint. This call is deliberately never made on a Public Host — see
// listOrgsPublic's doc comment for why it's unusable there.
func (a *Adapter) fetchAllOrgs(ctx context.Context, host forge.Host) ([]string, error) {
	stdout, err := a.runAPI(ctx, host, "organizations", "-f", "per_page=100")
	if err != nil {
		return nil, err
	}

	var entries []instanceOrgEntry
	if err := json.Unmarshal(stdout, &entries); err != nil {
		return nil, unmarshalError("organizations", err)
	}

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Login
	}
	return names, nil
}
