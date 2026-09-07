package github

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// listOrgsPublic implements Org discovery for a Public Host (ADR-0002). The
// instance-wide organizations endpoint is unusable here — it walks every organization
// on github.com in id order — so instead this unions two calls: the user's
// memberships (which yields Owner/Member) and a collaborator-affiliation repo probe
// (which yields Orgs the user has no membership in but can still see a repo in). A
// membership-derived Affiliation always wins over the collaborator probe for the same
// Org.
func (a *Adapter) listOrgsPublic(ctx context.Context, host forge.Host) ([]forge.Org, error) {
	memberships, err := a.fetchMemberships(ctx, host)
	if err != nil {
		return nil, err
	}
	collaboratorOrgs, err := a.fetchCollaboratorOrgs(ctx, host)
	if err != nil {
		return nil, err
	}

	byName := make(map[string]forge.Org, len(memberships)+len(collaboratorOrgs))
	for _, org := range memberships {
		byName[org.Name] = org
	}
	for _, name := range collaboratorOrgs {
		if _, exists := byName[name]; !exists {
			byName[name] = forge.Org{Name: name, Affiliation: forge.AffiliationCollaborator}
		}
	}

	orgs := make([]forge.Org, 0, len(byName))
	for _, org := range byName {
		orgs = append(orgs, org)
	}
	sort.Slice(orgs, func(i, j int) bool { return orgs[i].Name < orgs[j].Name })
	return orgs, nil
}

type membershipEntry struct {
	Organization struct {
		Login string `json:"login"`
	} `json:"organization"`
	Role string `json:"role"`
}

// fetchMemberships returns the Orgs the user is a member or owner of. GitHub's
// membership role is "admin" (organization owner) or "member" (regular member).
func (a *Adapter) fetchMemberships(ctx context.Context, host forge.Host) ([]forge.Org, error) {
	stdout, err := a.runAPI(ctx, host, "user/memberships/orgs", "-f", "per_page=100")
	if err != nil {
		return nil, err
	}

	var entries []membershipEntry
	if err := json.Unmarshal(stdout, &entries); err != nil {
		return nil, unmarshalError("user/memberships/orgs", err)
	}

	orgs := make([]forge.Org, 0, len(entries))
	for _, e := range entries {
		affiliation := forge.AffiliationMember
		if e.Role == "admin" {
			affiliation = forge.AffiliationOwner
		}
		orgs = append(orgs, forge.Org{Name: e.Organization.Login, Affiliation: affiliation})
	}
	return orgs, nil
}

type repoOwnerEntry struct {
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// fetchCollaboratorOrgs returns the distinct Org names of repos the user can see purely
// through outside-collaborator access — the case a membership-only view would miss
// entirely.
func (a *Adapter) fetchCollaboratorOrgs(ctx context.Context, host forge.Host) ([]string, error) {
	stdout, err := a.runAPI(ctx, host, "user/repos", "-f", "affiliation=collaborator", "-f", "per_page=100")
	if err != nil {
		return nil, err
	}

	var entries []repoOwnerEntry
	if err := json.Unmarshal(stdout, &entries); err != nil {
		return nil, unmarshalError("user/repos", err)
	}

	seen := make(map[string]bool, len(entries))
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !seen[e.Owner.Login] {
			seen[e.Owner.Login] = true
			names = append(names, e.Owner.Login)
		}
	}
	return names, nil
}

func unmarshalError(endpoint string, cause error) error {
	return &forge.Error{
		Kind:    forge.ErrKindPaneScoped,
		Message: fmt.Sprintf("could not parse the response from %s", endpoint),
		Err:     cause,
	}
}
