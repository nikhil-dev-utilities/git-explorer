package github

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestListRepos_MapsAllFields(t *testing.T) {
	host := forge.Host{Name: "github.com"}
	org := forge.Org{Name: "acme", Host: host}
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "orgs/acme/repos", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "org_repos.json"), ExitCode: 0},
	)

	a := newWithRunner(fr)
	repos, err := a.ListRepos(context.Background(), org)
	if err != nil {
		t.Fatalf("ListRepos() error = %v", err)
	}
	if len(repos) != 4 {
		t.Fatalf("got %d repos, want 4: %+v", len(repos), repos)
	}

	api := repos[0]
	if api.Name != "api" || api.Org != "acme" || api.Host != host {
		t.Errorf("repos[0] identity = %+v, want name=api org=acme host=%+v", api, host)
	}
	wantPushedAt := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	if !api.PushedAt.Equal(wantPushedAt) {
		t.Errorf("repos[0].PushedAt = %v, want %v", api.PushedAt, wantPushedAt)
	}
	if api.Archived || api.Fork {
		t.Errorf("repos[0] Archived/Fork = %v/%v, want false/false", api.Archived, api.Fork)
	}

	if !repos[2].Archived {
		t.Errorf("repos[2] (internal-tools) Archived = false, want true")
	}

	empty := repos[3]
	if !empty.Fork {
		t.Errorf("repos[3] (empty-repo) Fork = false, want true")
	}
	if !empty.PushedAt.IsZero() {
		t.Errorf("repos[3].PushedAt = %v, want the zero value for a null pushed_at", empty.PushedAt)
	}
}

func TestListRepos_AllThreeVisibilityValues(t *testing.T) {
	host := forge.Host{Name: "github.com"}
	org := forge.Org{Name: "acme", Host: host}
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "orgs/acme/repos", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "org_repos.json"), ExitCode: 0},
	)

	a := newWithRunner(fr)
	repos, err := a.ListRepos(context.Background(), org)
	if err != nil {
		t.Fatalf("ListRepos() error = %v", err)
	}

	want := map[string]forge.Visibility{
		"api":            forge.VisibilityPublic,
		"infra":          forge.VisibilityPrivate,
		"internal-tools": forge.VisibilityInternal,
		"empty-repo":     forge.VisibilityPublic,
	}
	for _, r := range repos {
		if r.Visibility != want[r.Name] {
			t.Errorf("repo %q Visibility = %v, want %v", r.Name, r.Visibility, want[r.Name])
		}
	}
}

func TestListRepos_StaysLazy(t *testing.T) {
	host := forge.Host{Name: "github.com", Kind: forge.HostPublic}
	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "github.com"}, runResult{ExitCode: 0})
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "user/memberships/orgs", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_memberships_orgs.json"), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "user/repos", "-f", "affiliation=collaborator", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_repos_collaborator.json"), ExitCode: 0},
	)

	a := newWithRunner(fr)
	ch, err := a.ListOrgs(context.Background(), host)
	if err != nil {
		t.Fatalf("ListOrgs() error = %v", err)
	}
	orgs := drainOrgs(t, ch)

	// Listing Orgs must not have triggered any repo listing. "user/repos" is the
	// legitimate collaborator probe ListOrgs itself uses; what must never appear is
	// a per-Org repos listing, e.g. "orgs/acme/repos" (the ListRepos endpoint).
	for _, call := range fr.snapshotCalls() {
		for _, arg := range call {
			if arg != "user/repos" && strings.HasSuffix(arg, "/repos") {
				t.Fatalf("ListOrgs triggered a repo-listing call, want it deferred until an Org is selected: %v", call)
			}
		}
	}

	// Now explicitly "select" the first Org and confirm exactly one repo-listing call
	// happens, for that Org specifically.
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "orgs/" + orgs[0].Name + "/repos", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte(`[]`), ExitCode: 0},
	)
	if _, err := a.ListRepos(context.Background(), orgs[0]); err != nil {
		t.Fatalf("ListRepos() error = %v", err)
	}

	repoCalls := 0
	for _, call := range fr.snapshotCalls() {
		for _, arg := range call {
			if strings.HasSuffix(arg, "/repos") && arg != "user/repos" {
				repoCalls++
			}
		}
	}
	if repoCalls != 1 {
		t.Fatalf("got %d repo-listing calls after selecting one Org, want exactly 1", repoCalls)
	}
}
