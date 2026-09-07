// Package forge defines the port through which the rest of git-explorer talks to a
// git-hosting Forge (GitHub in v1). Nothing outside this package's implementations
// (internal/forge/github, and any future internal/forge/<forge>) may reach past this
// interface — see ADR-0001.
package forge

import (
	"context"
	"time"
)

// HostKind distinguishes a Public Host (github.com), where Org discovery is limited to
// Orgs the user has some Affiliation with, from a Private Host (a self-managed
// instance), where every Org on the instance is listed. See ADR-0002.
type HostKind int

const (
	HostPublic HostKind = iota
	HostPrivate
)

// Host is a single addressable Forge installation.
type Host struct {
	// Name is the Host's address, e.g. "github.com" or "ghe.corp.internal".
	Name string
	Kind HostKind
	// Protocol is "ssh" or "https", used to build clone URLs for Repos on this Host.
	Protocol string
}

// Affiliation is the relationship between the user's credentials and an Org or Repo.
// The zero value, AffiliationNone, is itself a normal, displayable state on a Private
// Host — it is not an error or an absence of data.
type Affiliation int

const (
	AffiliationNone Affiliation = iota
	AffiliationCollaborator
	AffiliationMember
	AffiliationOwner
)

func (a Affiliation) String() string {
	switch a {
	case AffiliationOwner:
		return "owner"
	case AffiliationMember:
		return "member"
	case AffiliationCollaborator:
		return "collaborator"
	default:
		return "none"
	}
}

// Org is a namespace on a Host that owns Repos.
type Org struct {
	Name        string
	Affiliation Affiliation
}

// Visibility is a Repo's GitHub visibility. It has three real values — Internal is not a
// synonym for Private, it's a distinct value only meaningful on a Private Host.
type Visibility int

const (
	VisibilityPublic Visibility = iota
	VisibilityPrivate
	VisibilityInternal
)

func (v Visibility) String() string {
	switch v {
	case VisibilityPrivate:
		return "private"
	case VisibilityInternal:
		return "internal"
	default:
		return "public"
	}
}

// Repo is a single cloneable repository belonging to exactly one Org.
type Repo struct {
	Name       string
	Org        string
	Host       Host
	PushedAt   time.Time
	Archived   bool
	Fork       bool
	Visibility Visibility
}

// OrgPage is one batch of Orgs delivered by ListOrgs. Err, when non-nil, ends the stream
// with a pane-scoped failure — Orgs already delivered on earlier pages remain valid and
// must not be discarded by the caller.
type OrgPage struct {
	Orgs []Org
	Err  error
}

// Forge is the only interface the rest of git-explorer depends on to talk to a Host.
type Forge interface {
	// ListOrgs streams Orgs for the given Host. The returned error is fatal — returned
	// before any page is sent — and distinct from a per-page error carried on OrgPage.
	ListOrgs(ctx context.Context, host Host) (<-chan OrgPage, error)

	// ListRepos is called lazily, only once an Org has been selected.
	ListRepos(ctx context.Context, org Org) ([]Repo, error)

	// CloneURL builds the clone URL for repo, respecting its Host's configured
	// Protocol.
	CloneURL(repo Repo) string
}
