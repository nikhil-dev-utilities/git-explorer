package github

import (
	"context"
	"strings"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func registerPrivateHostFixtures(t *testing.T, fr *fakeRunner, hostName string) {
	t.Helper()
	fr.on(
		[]string{"api", "--hostname", hostName, "organizations", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "organizations.json"), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", hostName, "user/memberships/orgs", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_memberships_orgs.json"), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", hostName, "user/repos", "-f", "affiliation=collaborator", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_repos_collaborator.json"), ExitCode: 0},
	)
}

func TestListOrgsPrivate_ReturnsEveryOrgBadgedByAffiliation(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal", Kind: forge.HostPrivate}
	fr := newFakeRunner()
	registerPrivateHostFixtures(t, fr, "ghe.corp.internal")

	a := newWithRunner(fr)
	orgs, err := a.listOrgsPrivate(context.Background(), host)
	if err != nil {
		t.Fatalf("listOrgsPrivate() error = %v", err)
	}

	// organizations.json lists acme, widgets-inc, globex, initech — all four must be
	// present, including initech (which appears in neither membership nor
	// collaborator fixtures, so it must come back AffiliationNone, not be dropped).
	want := map[string]forge.Affiliation{
		"acme":        forge.AffiliationOwner,        // admin in user/memberships/orgs
		"widgets-inc": forge.AffiliationMember,       // member in user/memberships/orgs
		"globex":      forge.AffiliationCollaborator, // collaborator-probe only
		"initech":     forge.AffiliationNone,         // in neither fixture
	}
	if len(orgs) != len(want) {
		t.Fatalf("got %d orgs, want %d: %+v", len(orgs), len(want), orgs)
	}
	for _, org := range orgs {
		wantAff, ok := want[org.Name]
		if !ok {
			t.Errorf("unexpected org %q in result", org.Name)
			continue
		}
		if org.Affiliation != wantAff {
			t.Errorf("org %q Affiliation = %v, want %v", org.Name, org.Affiliation, wantAff)
		}
	}
}

func TestListOrgsPrivate_DoesNotFetchRepos(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal", Kind: forge.HostPrivate}
	fr := newFakeRunner()
	registerPrivateHostFixtures(t, fr, "ghe.corp.internal")

	a := newWithRunner(fr)
	if _, err := a.listOrgsPrivate(context.Background(), host); err != nil {
		t.Fatalf("listOrgsPrivate() error = %v", err)
	}

	// "user/repos" is the legitimate collaborator probe; what must never appear is a
	// per-Org repos listing, e.g. "orgs/acme/repos" (the ListRepos endpoint).
	for _, call := range fr.calls {
		for _, arg := range call {
			if arg != "user/repos" && strings.HasSuffix(arg, "/repos") {
				t.Fatalf("listOrgsPrivate fetched an Org's repos, want it to stay lazy: %v", call)
			}
		}
	}
}

func TestAdapter_ListOrgs_DispatchesToPrivateHostPath(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal", Kind: forge.HostPrivate}
	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "ghe.corp.internal"}, runResult{ExitCode: 0})
	registerPrivateHostFixtures(t, fr, "ghe.corp.internal")

	a := newWithRunner(fr)
	ch, err := a.ListOrgs(context.Background(), host)
	if err != nil {
		t.Fatalf("ListOrgs() error = %v", err)
	}

	var orgs []forge.Org
	for page := range ch {
		if page.Err != nil {
			t.Fatalf("page.Err = %v, want nil", page.Err)
		}
		orgs = append(orgs, page.Orgs...)
	}
	if len(orgs) != 4 {
		t.Fatalf("got %d orgs, want 4 (the full instance listing): %+v", len(orgs), orgs)
	}
}
