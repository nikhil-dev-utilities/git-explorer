package github

import (
	"context"
	"os"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return data
}

func TestListOrgsPublic_UnionsMembershipAndCollaborator(t *testing.T) {
	host := forge.Host{Name: "github.com", Kind: forge.HostPublic}
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "user/memberships/orgs", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_memberships_orgs.json"), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "user/repos", "-f", "affiliation=collaborator", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_repos_collaborator.json"), ExitCode: 0},
	)

	a := newWithRunner(fr)
	orgs, err := a.listOrgsPublic(context.Background(), host)
	if err != nil {
		t.Fatalf("listOrgsPublic() error = %v", err)
	}

	want := []forge.Org{
		{Name: "acme", Host: host, Affiliation: forge.AffiliationOwner},
		{Name: "globex", Host: host, Affiliation: forge.AffiliationCollaborator},
		{Name: "widgets-inc", Host: host, Affiliation: forge.AffiliationMember},
	}
	if len(orgs) != len(want) {
		t.Fatalf("got %d orgs, want %d: %+v", len(orgs), len(want), orgs)
	}
	for i, w := range want {
		if orgs[i] != w {
			t.Errorf("orgs[%d] = %+v, want %+v", i, orgs[i], w)
		}
	}
}

func TestListOrgsPublic_MembershipWinsOverCollaboratorProbe(t *testing.T) {
	// acme appears in both fixtures: as an admin membership and as a collaborator
	// repo owner. The result must keep it as Owner, never downgrade it to
	// Collaborator because of the probe.
	host := forge.Host{Name: "github.com"}
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "user/memberships/orgs", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_memberships_orgs.json"), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "user/repos", "-f", "affiliation=collaborator", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_repos_collaborator.json"), ExitCode: 0},
	)

	a := newWithRunner(fr)
	orgs, err := a.listOrgsPublic(context.Background(), host)
	if err != nil {
		t.Fatalf("listOrgsPublic() error = %v", err)
	}

	for _, o := range orgs {
		if o.Name == "acme" && o.Affiliation != forge.AffiliationOwner {
			t.Fatalf("acme Affiliation = %v, want AffiliationOwner (membership must win)", o.Affiliation)
		}
	}
}

func TestListOrgsPublic_APIFailureIsPaneScoped(t *testing.T) {
	host := forge.Host{Name: "github.com"}
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "user/memberships/orgs", "-f", "per_page=100"},
		runResult{Stderr: []byte("HTTP 500: internal error"), ExitCode: 1},
	)

	a := newWithRunner(fr)
	_, err := a.listOrgsPublic(context.Background(), host)
	if err == nil {
		t.Fatal("listOrgsPublic() error = nil, want an error")
	}
	fe, ok := err.(*forge.Error)
	if !ok {
		t.Fatalf("error = %v (%T), want a *forge.Error", err, err)
	}
	if fe.Kind != forge.ErrKindPaneScoped {
		t.Fatalf("Kind = %v, want ErrKindPaneScoped", fe.Kind)
	}
}
