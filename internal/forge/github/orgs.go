package github

import (
	"context"
	"log/slog"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// ListOrgs checks authentication for host first — failing fast with a specific message
// rather than surfacing as a generic failure from whichever call happens to run first —
// then dispatches by Host kind per ADR-0002: the two paths genuinely diverge, not just
// in parameters.
//
// The Public Host path is small and bounded (the user's own memberships plus a
// collaborator probe), so it's fetched synchronously and delivered as a single page.
// The Private Host path can enumerate an entire instance, so it streams: a goroutine
// paginates in the background and the caller can read pages as they arrive rather than
// waiting for the whole thing.
func (a *Adapter) ListOrgs(ctx context.Context, host forge.Host) (<-chan forge.OrgPage, error) {
	slog.InfoContext(ctx, "listing orgs", "host", host.Name, "kind", host.Kind)

	if err := a.checkAuth(ctx, host); err != nil {
		return nil, err
	}

	if host.Kind == forge.HostPrivate {
		out := make(chan forge.OrgPage)
		go a.streamOrgsPrivate(ctx, host, out)
		return out, nil
	}

	orgs, err := a.listOrgsPublic(ctx, host)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "listed orgs", "host", host.Name, "count", len(orgs))

	page := make(chan forge.OrgPage, 1)
	page <- forge.OrgPage{Orgs: orgs}
	close(page)
	return page, nil
}
