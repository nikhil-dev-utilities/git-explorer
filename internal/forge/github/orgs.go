package github

import (
	"context"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// ListOrgs checks authentication for host first — failing fast with a specific message
// rather than surfacing as a generic failure from whichever call happens to run first —
// then dispatches by Host kind per ADR-0002: the two paths genuinely diverge, not just
// in parameters.
//
// The result is always delivered as a single page for now. Real progressive, multi-page
// delivery is a later slice; ListOrgs already has its final streaming signature so that
// slice is additive rather than an interface break.
func (a *Adapter) ListOrgs(ctx context.Context, host forge.Host) (<-chan forge.OrgPage, error) {
	if err := a.checkAuth(ctx, host); err != nil {
		return nil, err
	}

	var orgs []forge.Org
	var err error
	if host.Kind == forge.HostPrivate {
		orgs, err = a.listOrgsPrivate(ctx, host)
	} else {
		orgs, err = a.listOrgsPublic(ctx, host)
	}
	if err != nil {
		return nil, err
	}

	page := make(chan forge.OrgPage, 1)
	page <- forge.OrgPage{Orgs: orgs}
	close(page)
	return page, nil
}
