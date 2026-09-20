package ui

import (
	"context"
	"sync"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

type fakeForge struct {
	mu        sync.Mutex
	pages     map[string][]forge.OrgPage // by Host name
	orgsErr   error
	errByHost map[string]error
	repos     map[string][]forge.Repo // by Org name
	reposErr  error

	orgCalls  []string
	repoCalls []string
}

func (f *fakeForge) ListOrgs(_ context.Context, host forge.Host) (<-chan forge.OrgPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orgCalls = append(f.orgCalls, host.Name)
	if e := f.errByHost[host.Name]; e != nil {
		return nil, e
	}
	if f.orgsErr != nil {
		return nil, f.orgsErr
	}
	pages := f.pages[host.Name]
	ch := make(chan forge.OrgPage, len(pages))
	for _, p := range pages {
		ch <- p
	}
	close(ch)
	return ch, nil
}

func (f *fakeForge) ListRepos(_ context.Context, org forge.Org) ([]forge.Repo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repoCalls = append(f.repoCalls, org.Name)
	if f.reposErr != nil {
		return nil, f.reposErr
	}
	return f.repos[org.Name], nil
}

func (f *fakeForge) CloneURL(r forge.Repo) string { return "fake://" + r.Org + "/" + r.Name }

var _ forge.Forge = (*fakeForge)(nil)
