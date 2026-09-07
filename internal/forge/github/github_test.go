package github

import (
	"context"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestAdapter_ListOrgs_PublicHost(t *testing.T) {
	host := forge.Host{Name: "github.com", Kind: forge.HostPublic}
	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "github.com"}, runResult{
		Stdout: []byte("canary-token-must-never-leak"), ExitCode: 0,
	})
	fr.on(
		[]string{"api", "--hostname", "github.com", "user/memberships/orgs", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_memberships_orgs.json"), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", "github.com", "user/repos", "-f", "affiliation=collaborator", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_repos_collaborator.json"), ExitCode: 0},
	)

	a := newWithRunner(fr)
	ch, err := a.ListOrgs(context.Background(), host)
	if err != nil {
		t.Fatalf("ListOrgs() error = %v", err)
	}

	var pages []forge.OrgPage
	for page := range ch {
		pages = append(pages, page)
	}
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want exactly 1 for this slice", len(pages))
	}
	if pages[0].Err != nil {
		t.Fatalf("page.Err = %v, want nil", pages[0].Err)
	}
	if len(pages[0].Orgs) != 3 {
		t.Fatalf("got %d orgs, want 3: %+v", len(pages[0].Orgs), pages[0].Orgs)
	}
}

func TestAdapter_ListOrgs_NotAuthenticatedFailsBeforeAnyDataCall(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal"}
	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "ghe.corp.internal"}, runResult{
		Stderr: []byte("not logged in"), ExitCode: 1,
	})
	// Deliberately no responses registered for any `api` call — if ListOrgs called
	// one anyway, the fake would return an error for "no response configured" and
	// this test would still pass for the wrong reason, so we assert on fr.calls too.

	a := newWithRunner(fr)
	_, err := a.ListOrgs(context.Background(), host)
	if err == nil {
		t.Fatal("ListOrgs() error = nil, want an error")
	}
	fe, ok := err.(*forge.Error)
	if !ok {
		t.Fatalf("error = %v (%T), want a *forge.Error", err, err)
	}
	if fe.Kind != forge.ErrKindFatal {
		t.Fatalf("Kind = %v, want ErrKindFatal", fe.Kind)
	}
	if len(fr.calls) != 1 {
		t.Fatalf("gh was invoked %d times, want exactly 1 (the auth check only): %v", len(fr.calls), fr.calls)
	}
}
