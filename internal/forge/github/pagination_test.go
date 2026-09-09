package github

import (
	"context"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// fetchAllPagesEntry matches orgsJSON's shape (a "login" field), reused here to build
// test fixtures without hand-writing large JSON arrays.
type fetchAllPagesEntry struct {
	Name string `json:"login"`
}

func TestFetchAllPages_StopsAfterAShortFirstPage(t *testing.T) {
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "things", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: orgsJSON(t, namesN("thing", 3), 1), ExitCode: 0},
	)
	a := newWithRunner(fr)

	got, err := fetchAllPages[fetchAllPagesEntry](context.Background(), a, forge.Host{Name: "github.com"}, "things")
	if err != nil {
		t.Fatalf("fetchAllPages() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3 — a short first page must not trigger a second request", len(got))
	}
	if calls := len(fr.snapshotCalls()); calls != 1 {
		t.Fatalf("got %d calls, want exactly 1", calls)
	}
}

// This is the regression this whole file exists to prevent: fetchMemberships,
// fetchCollaboratorOrgs, and ListRepos all used to fetch exactly one page and
// silently drop everything past the first 100 results, with no error or indication
// to the caller. A full first page must now trigger a second request.
func TestFetchAllPages_ContinuesPastAFullPageAndConcatenates(t *testing.T) {
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "things", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: orgsJSON(t, namesN("thing", defaultPerPage), 1), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "things", "-f", "page=2", "-f", "per_page=100"},
		runResult{Stdout: orgsJSON(t, namesN("more", 7), 1), ExitCode: 0},
	)
	a := newWithRunner(fr)

	got, err := fetchAllPages[fetchAllPagesEntry](context.Background(), a, forge.Host{Name: "github.com"}, "things")
	if err != nil {
		t.Fatalf("fetchAllPages() error = %v", err)
	}
	want := defaultPerPage + 7
	if len(got) != want {
		t.Fatalf("got %d entries, want %d (a full page plus a short second page)", len(got), want)
	}
	if got[0].Name != "thing-0" || got[defaultPerPage].Name != "more-0" {
		t.Fatalf("pages were not concatenated in order: first=%q, entry[%d]=%q", got[0].Name, defaultPerPage, got[defaultPerPage].Name)
	}
}

func TestFetchAllPages_PropagatesAFailureOnALaterPage(t *testing.T) {
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "things", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: orgsJSON(t, namesN("thing", defaultPerPage), 1), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "things", "-f", "page=2", "-f", "per_page=100"},
		runResult{Stderr: []byte("boom"), ExitCode: 1},
	)
	a := newWithRunner(fr)

	_, err := fetchAllPages[fetchAllPagesEntry](context.Background(), a, forge.Host{Name: "github.com"}, "things")
	if err == nil {
		t.Fatal("fetchAllPages() error = nil, want the second page's failure")
	}
}

// TestListRepos_PaginatesPastTheFirstHundred is the caller-level proof that the
// pagination gap is actually closed for the endpoint the user hit it on: an Org with
// more than 100 repos used to silently lose everything past the first page.
func TestListRepos_PaginatesPastTheFirstHundred(t *testing.T) {
	host := forge.Host{Name: "github.com"}
	org := forge.Org{Name: "acme", Host: host}
	fr := newFakeRunner()
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "orgs/acme/repos", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: orgsJSON(t, namesN("repo", defaultPerPage), 1), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", "github.com", "-X", "GET", "orgs/acme/repos", "-f", "page=2", "-f", "per_page=100"},
		runResult{Stdout: orgsJSON(t, namesN("repo-page2", 12), 1), ExitCode: 0},
	)
	a := newWithRunner(fr)

	repos, err := a.ListRepos(context.Background(), org)
	if err != nil {
		t.Fatalf("ListRepos() error = %v", err)
	}
	if want := defaultPerPage + 12; len(repos) != want {
		t.Fatalf("got %d repos, want %d — repos past the first page must not be dropped", len(repos), want)
	}
}
