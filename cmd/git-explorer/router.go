package main

import (
	"context"
	"fmt"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// forgeRouter is the one forge.Forge the UI sees. It dispatches each call to the adapter
// named by the Host already carried on the call's argument — see ADR-0010.
type forgeRouter map[string]forge.Forge

func (r forgeRouter) ListOrgs(ctx context.Context, host forge.Host) (<-chan forge.OrgPage, error) {
	f, err := r.lookup(host)
	if err != nil {
		return nil, err
	}
	return f.ListOrgs(ctx, host)
}

func (r forgeRouter) ListRepos(ctx context.Context, org forge.Org) ([]forge.Repo, error) {
	f, err := r.lookup(org.Host)
	if err != nil {
		return nil, err
	}
	return f.ListRepos(ctx, org)
}

// CloneURL returns "" for a Host with no adapter. That cannot happen in practice: the
// same Host's ListOrgs already failed, so it has no Repos to clone.
func (r forgeRouter) CloneURL(repo forge.Repo) string {
	f, err := r.lookup(repo.Host)
	if err != nil {
		return ""
	}
	return f.CloneURL(repo)
}

func (r forgeRouter) lookup(host forge.Host) (forge.Forge, error) {
	if f, ok := r[host.Forge]; ok {
		return f, nil
	}
	return nil, &forge.Error{
		Kind:    forge.ErrKindFatal,
		Message: fmt.Sprintf("Host %s uses forge %q, which this build of git-explorer does not support.", host.Name, host.Forge),
	}
}
