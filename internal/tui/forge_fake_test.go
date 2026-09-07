package tui

import (
	"context"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// fakeForge is the seam every test in this package injects instead of a real
// forge.Forge — this package never imports internal/forge/github, so a fake is the
// only way its tests exercise ListOrgs/ListRepos/CloneURL at all.
type fakeForge struct {
	orgPages []forge.OrgPage
	orgsErr  error // returned synchronously by ListOrgs, if set
	// orgsGate, if set, is received from before ListOrgs returns anything — lets a
	// test observe the Model's state deterministically before any page has arrived,
	// rather than racing an instantly-resolving fake.
	orgsGate <-chan struct{}
	// orgPagesByHost, if set, overrides orgPages on a per-Host-name basis — for
	// host-switching tests, which need different Orgs to appear after switching.
	orgPagesByHost map[string][]forge.OrgPage
	// orgPagesSequence, if set, overrides orgPages per successive ListOrgs call —
	// index 0 for the first call, index 1 for the second, and so on, holding at the
	// last entry once exhausted. For retry tests: a call that fails followed by one
	// that succeeds.
	orgPagesSequence [][]forge.OrgPage
	listOrgsCalls    []forge.Host

	repos    map[string][]forge.Repo // keyed by Org name
	reposErr error

	listReposCalls []forge.Org
}

func (f *fakeForge) ListOrgs(_ context.Context, host forge.Host) (<-chan forge.OrgPage, error) {
	callIndex := len(f.listOrgsCalls)
	f.listOrgsCalls = append(f.listOrgsCalls, host)
	if f.orgsGate != nil {
		<-f.orgsGate
	}
	if f.orgsErr != nil {
		return nil, f.orgsErr
	}
	pages := f.orgPages
	switch {
	case f.orgPagesSequence != nil:
		i := callIndex
		if i >= len(f.orgPagesSequence) {
			i = len(f.orgPagesSequence) - 1
		}
		pages = f.orgPagesSequence[i]
	case f.orgPagesByHost != nil:
		pages = f.orgPagesByHost[host.Name]
	}
	ch := make(chan forge.OrgPage, len(pages))
	for _, p := range pages {
		ch <- p
	}
	close(ch)
	return ch, nil
}

func (f *fakeForge) ListRepos(_ context.Context, org forge.Org) ([]forge.Repo, error) {
	f.listReposCalls = append(f.listReposCalls, org)
	if f.reposErr != nil {
		return nil, f.reposErr
	}
	return f.repos[org.Name], nil
}

func (f *fakeForge) CloneURL(repo forge.Repo) string {
	return "fake://" + repo.Org + "/" + repo.Name
}

var _ forge.Forge = (*fakeForge)(nil)
