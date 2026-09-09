package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// orgsJSON builds the JSON body the organizations endpoint would return for the given
// Org names — used instead of a static testdata file so pagination tests can build a
// page of any size (including a full orgsPerPage page) without hand-writing large
// fixtures. IDs are assigned sequentially from startID (GET /organizations is
// ID-ordered ascending), so a test can compute the since cursor the next page's fixture
// must be registered under as startID+len(names)-1.
func orgsJSON(t *testing.T, names []string, startID int64) []byte {
	t.Helper()
	entries := make([]map[string]any, len(names))
	for i, n := range names {
		entries[i] = map[string]any{"login": n, "id": startID + int64(i)}
	}
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshaling org fixture: %v", err)
	}
	return data
}

func namesN(prefix string, n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("%s-%d", prefix, i)
	}
	return names
}

func registerPrivateHostFixtures(t *testing.T, fr *fakeRunner, hostName string) {
	t.Helper()
	fr.on(
		[]string{"api", "--hostname", hostName, "-X", "GET", "organizations", "-f", "since=0", "-f", fmt.Sprintf("per_page=%d", orgsPerPage)},
		runResult{Stdout: readFixture(t, "organizations.json"), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", hostName, "-X", "GET", "user/memberships/orgs", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_memberships_orgs.json"), ExitCode: 0},
	)
	fr.on(
		[]string{"api", "--hostname", hostName, "-X", "GET", "user/repos", "-f", "affiliation=collaborator", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: readFixture(t, "user_repos_collaborator.json"), ExitCode: 0},
	)
}

// drainOrgs collects every Org from every page of ch, failing the test on the first
// page-level error.
func drainOrgs(t *testing.T, ch <-chan forge.OrgPage) []forge.Org {
	t.Helper()
	var orgs []forge.Org
	for page := range ch {
		if page.Err != nil {
			t.Fatalf("unexpected page.Err = %v", page.Err)
		}
		orgs = append(orgs, page.Orgs...)
	}
	return orgs
}

func TestAdapter_ListOrgs_PrivateHost_ReturnsEveryOrgBadgedByAffiliation(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal", Kind: forge.HostPrivate}
	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "ghe.corp.internal"}, runResult{ExitCode: 0})
	registerPrivateHostFixtures(t, fr, "ghe.corp.internal")

	a := newWithRunner(fr)
	ch, err := a.ListOrgs(context.Background(), host)
	if err != nil {
		t.Fatalf("ListOrgs() error = %v", err)
	}
	orgs := drainOrgs(t, ch)

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

func TestAdapter_ListOrgs_PrivateHost_DoesNotFetchRepos(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal", Kind: forge.HostPrivate}
	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "ghe.corp.internal"}, runResult{ExitCode: 0})
	registerPrivateHostFixtures(t, fr, "ghe.corp.internal")

	a := newWithRunner(fr)
	ch, err := a.ListOrgs(context.Background(), host)
	if err != nil {
		t.Fatalf("ListOrgs() error = %v", err)
	}
	drainOrgs(t, ch)

	// "user/repos" is the legitimate collaborator probe; what must never appear is a
	// per-Org repos listing, e.g. "orgs/acme/repos" (the ListRepos endpoint).
	for _, call := range fr.snapshotCalls() {
		for _, arg := range call {
			if arg != "user/repos" && strings.HasSuffix(arg, "/repos") {
				t.Fatalf("fetched an Org's repos, want listing Orgs to stay lazy: %v", call)
			}
		}
	}
}

func TestStreamOrgsPrivate_PaginatesUntilAShortPage(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal", Kind: forge.HostPrivate}
	page1Names := namesN("org", orgsPerPage) // a full page: more must be requested
	page2Names := []string{"org-last-a", "org-last-b"}

	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "ghe.corp.internal"}, runResult{ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "user/memberships/orgs", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte(`[]`), ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "user/repos", "-f", "affiliation=collaborator", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte(`[]`), ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "organizations", "-f", "since=0", "-f", fmt.Sprintf("per_page=%d", orgsPerPage)},
		runResult{Stdout: orgsJSON(t, page1Names, 1), ExitCode: 0})
	// The since cursor for the second request must be the last ID from page 1
	// (1 + orgsPerPage - 1), never a page number — this is the regression this test
	// exists to catch: passing page=2 here (as before the fix) is silently ignored by
	// GET /organizations, which would just return page 1's data again forever.
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "organizations", "-f", fmt.Sprintf("since=%d", orgsPerPage), "-f", fmt.Sprintf("per_page=%d", orgsPerPage)},
		runResult{Stdout: orgsJSON(t, page2Names, orgsPerPage+1), ExitCode: 0})

	a := newWithRunner(fr)
	ch, err := a.ListOrgs(context.Background(), host)
	if err != nil {
		t.Fatalf("ListOrgs() error = %v", err)
	}

	var pages [][]forge.Org
	for page := range ch {
		if page.Err != nil {
			t.Fatalf("unexpected page.Err = %v", page.Err)
		}
		pages = append(pages, page.Orgs)
	}

	if len(pages) != 2 {
		t.Fatalf("got %d pages, want exactly 2 (a full page then a short one)", len(pages))
	}
	if len(pages[0]) != orgsPerPage {
		t.Fatalf("page 1 has %d orgs, want %d", len(pages[0]), orgsPerPage)
	}
	if len(pages[1]) != len(page2Names) {
		t.Fatalf("page 2 has %d orgs, want %d", len(pages[1]), len(page2Names))
	}
	if pages[0][0].Name != "org-0" {
		t.Fatalf("page 1 does not contain page 1's data (got %q first)", pages[0][0].Name)
	}
	if pages[1][0].Name != "org-last-a" {
		t.Fatalf("page 2 does not contain page 2's data (got %q first)", pages[1][0].Name)
	}

	// This is the exact shape of the bug reported live against a real GHE instance:
	// GET /organizations paginates via since (an org ID cursor), not page — passing
	// page=N (silently ignored by that endpoint) made every request identical, so the
	// same "since=0" call would have been sent again and again, forever, instead of
	// exactly once.
	since0Calls := 0
	for _, call := range fr.snapshotCalls() {
		if contains(call, "organizations") && contains(call, "since=0") {
			since0Calls++
		}
	}
	if since0Calls != 1 {
		t.Fatalf("the since=0 request was made %d times, want exactly 1 — the cursor must advance, not repeat", since0Calls)
	}
}

func TestStreamOrgsPrivate_ObservesEarlyPageBeforeLaterPageRequested(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal", Kind: forge.HostPrivate}
	page1Names := namesN("org", orgsPerPage)
	release := make(chan struct{})

	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "ghe.corp.internal"}, runResult{ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "user/memberships/orgs", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte(`[]`), ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "user/repos", "-f", "affiliation=collaborator", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte(`[]`), ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "organizations", "-f", "since=0", "-f", fmt.Sprintf("per_page=%d", orgsPerPage)},
		runResult{Stdout: orgsJSON(t, page1Names, 1), ExitCode: 0})
	// page 2's call is gated: it will not be recorded in fr.calls, nor return, until
	// this test explicitly closes `release` — proving deterministically (not by
	// timing) that it has not been requested yet at the point we check.
	fr.onGated(
		[]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "organizations", "-f", fmt.Sprintf("since=%d", orgsPerPage), "-f", fmt.Sprintf("per_page=%d", orgsPerPage)},
		runResult{Stdout: orgsJSON(t, []string{"org-last"}, orgsPerPage+1), ExitCode: 0},
		release,
	)

	a := newWithRunner(fr)
	ch, err := a.ListOrgs(context.Background(), host)
	if err != nil {
		t.Fatalf("ListOrgs() error = %v", err)
	}

	page1 := <-ch
	if page1.Err != nil {
		t.Fatalf("page1.Err = %v, want nil", page1.Err)
	}
	if len(page1.Orgs) != orgsPerPage {
		t.Fatalf("page1 has %d orgs, want %d", len(page1.Orgs), orgsPerPage)
	}

	for _, call := range fr.snapshotCalls() {
		if len(call) > 0 && contains(call, fmt.Sprintf("since=%d", orgsPerPage)) {
			t.Fatal("page 2 was requested before the test released it")
		}
	}

	close(release)

	page2 := <-ch
	if page2.Err != nil {
		t.Fatalf("page2.Err = %v, want nil", page2.Err)
	}
	if len(page2.Orgs) != 1 || page2.Orgs[0].Name != "org-last" {
		t.Fatalf("page2.Orgs = %+v, want [org-last]", page2.Orgs)
	}

	if _, ok := <-ch; ok {
		t.Fatal("channel produced a third page, want it closed after the short page")
	}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func TestStreamOrgsPrivate_RateLimitIsTransientWithRetryAfter(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal", Kind: forge.HostPrivate}
	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "ghe.corp.internal"}, runResult{ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "user/memberships/orgs", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte(`[]`), ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "user/repos", "-f", "affiliation=collaborator", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte(`[]`), ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "organizations", "-f", "since=0", "-f", fmt.Sprintf("per_page=%d", orgsPerPage)},
		runResult{
			Stderr:   []byte("gh: API rate limit exceeded for user ID 123. (HTTP 403)\nRetry-After: 45"),
			ExitCode: 1,
		})

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
		t.Fatalf("got %d pages, want exactly 1 (the rate-limit failure)", len(pages))
	}
	if pages[0].Err == nil {
		t.Fatal("pages[0].Err = nil, want the rate-limit error")
	}
	fe, ok := pages[0].Err.(*forge.Error)
	if !ok {
		t.Fatalf("Err = %v (%T), want a *forge.Error", pages[0].Err, pages[0].Err)
	}
	if fe.Kind != forge.ErrKindTransient {
		t.Fatalf("Kind = %v, want ErrKindTransient", fe.Kind)
	}
	if fe.RetryAfter != 45*time.Second {
		t.Fatalf("RetryAfter = %v, want 45s", fe.RetryAfter)
	}
}

func TestStreamOrgsPrivate_LaterPageFailurePreservesEarlierPages(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal", Kind: forge.HostPrivate}
	page1Names := namesN("org", orgsPerPage)

	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "ghe.corp.internal"}, runResult{ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "user/memberships/orgs", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte(`[]`), ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "user/repos", "-f", "affiliation=collaborator", "-f", "page=1", "-f", "per_page=100"},
		runResult{Stdout: []byte(`[]`), ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "organizations", "-f", "since=0", "-f", fmt.Sprintf("per_page=%d", orgsPerPage)},
		runResult{Stdout: orgsJSON(t, page1Names, 1), ExitCode: 0})
	fr.on([]string{"api", "--hostname", "ghe.corp.internal", "-X", "GET", "organizations", "-f", fmt.Sprintf("since=%d", orgsPerPage), "-f", fmt.Sprintf("per_page=%d", orgsPerPage)},
		runResult{Stderr: []byte("HTTP 500: internal error"), ExitCode: 1})

	a := newWithRunner(fr)
	ch, err := a.ListOrgs(context.Background(), host)
	if err != nil {
		t.Fatalf("ListOrgs() error = %v", err)
	}

	page1 := <-ch
	if page1.Err != nil {
		t.Fatalf("page1.Err = %v, want nil", page1.Err)
	}
	if len(page1.Orgs) != orgsPerPage {
		t.Fatalf("page1 has %d orgs, want %d — it must survive the later failure intact", len(page1.Orgs), orgsPerPage)
	}

	page2 := <-ch
	if page2.Err == nil {
		t.Fatal("page2.Err = nil, want the page-2 failure")
	}
	fe, ok := page2.Err.(*forge.Error)
	if !ok {
		t.Fatalf("Err = %v (%T), want a *forge.Error", page2.Err, page2.Err)
	}
	if fe.Kind != forge.ErrKindPaneScoped {
		t.Fatalf("Kind = %v, want ErrKindPaneScoped", fe.Kind)
	}

	if _, ok := <-ch; ok {
		t.Fatal("channel produced a third value, want it closed after the failure")
	}
}
