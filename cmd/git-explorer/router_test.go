package main

import (
	"context"
	"errors"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// namedForge reports its own name from every call, so a test can see which adapter
// the router dispatched to.
type namedForge string

func (n namedForge) ListOrgs(ctx context.Context, host forge.Host) (<-chan forge.OrgPage, error) {
	ch := make(chan forge.OrgPage, 1)
	ch <- forge.OrgPage{Orgs: []forge.Org{{Name: string(n)}}}
	close(ch)
	return ch, nil
}
func (n namedForge) ListRepos(ctx context.Context, org forge.Org) ([]forge.Repo, error) {
	return []forge.Repo{{Name: string(n)}}, nil
}
func (n namedForge) CloneURL(repo forge.Repo) string { return string(n) }

func TestForgeRouter_DispatchesOnTheHostsForge(t *testing.T) {
	r := forgeRouter{"github": namedForge("gh"), "bitbucket": namedForge("bb")}
	bb := forge.Host{Name: "bitbucket.org", Forge: "bitbucket"}

	ch, err := r.ListOrgs(context.Background(), bb)
	if err != nil {
		t.Fatal(err)
	}
	if page := <-ch; page.Orgs[0].Name != "bb" {
		t.Errorf("ListOrgs went to %q, want bb", page.Orgs[0].Name)
	}
	repos, _ := r.ListRepos(context.Background(), forge.Org{Host: bb})
	if repos[0].Name != "bb" {
		t.Errorf("ListRepos went to %q, want bb", repos[0].Name)
	}
	if got := r.CloneURL(forge.Repo{Host: forge.Host{Forge: "github"}}); got != "gh" {
		t.Errorf("CloneURL went to %q, want gh", got)
	}
}

func TestForgeRouter_UnknownForgeIsFatal(t *testing.T) {
	r := forgeRouter{"github": namedForge("gh")}
	_, err := r.ListOrgs(context.Background(), forge.Host{Name: "bitbucket.org", Forge: "bitbucket"})

	var fe *forge.Error
	if !errors.As(err, &fe) || fe.Kind != forge.ErrKindFatal {
		t.Errorf("ListOrgs error = %v, want a Fatal *forge.Error", err)
	}
	if got := r.CloneURL(forge.Repo{Host: forge.Host{Forge: "bitbucket"}}); got != "" {
		t.Errorf("CloneURL = %q, want empty", got)
	}
}
