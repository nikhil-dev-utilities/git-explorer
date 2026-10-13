package bitbucket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

var cloud = forge.Host{Name: "bitbucket.org", Forge: "bitbucket", Kind: forge.HostPublic, Protocol: "https"}

// newTestAdapter serves routes from an httptest server and counts credential lookups.
func newTestAdapter(t *testing.T, routes map[string]http.HandlerFunc) (*Adapter, *int) {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, h := range routes {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if u, p, ok := r.BasicAuth(); !ok || u != "me@example.com" || p != secret {
				t.Errorf("%s: missing or wrong basic auth", r.URL.Path)
			}
			h(w, r)
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	lookups := 0
	a := &Adapter{
		http:    srv.Client(),
		baseURL: srv.URL + "/2.0",
		dcBase:  func(string) string { return srv.URL + "/rest/api/1.0" },
		getenv:  func(string) string { return "" },
		fill: func(context.Context, string) (credential, bool, error) {
			lookups++
			return credential{username: "me@example.com", password: secret}, true, nil
		},
	}
	return a, &lookups
}

func TestListOrgs_PaginatesAndMapsAffiliation(t *testing.T) {
	var a *Adapter
	a, lookups := newTestAdapter(t, map[string]http.HandlerFunc{
		"/2.0/user/workspaces": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") == "2" {
				fmt.Fprint(w, `{"values":[{"administrator":false,"workspace":{"slug":"team"}}]}`)
				return
			}
			fmt.Fprintf(w, `{"values":[{"administrator":true,"workspace":{"slug":"me"}}],"next":%q}`,
				a.baseURL+"/user/workspaces?page=2")
		},
	})

	ch, err := a.ListOrgs(context.Background(), cloud)
	if err != nil {
		t.Fatal(err)
	}
	orgs := (<-ch).Orgs
	if len(orgs) != 2 || orgs[0].Name != "me" || orgs[0].Affiliation != forge.AffiliationOwner ||
		orgs[1].Name != "team" || orgs[1].Affiliation != forge.AffiliationMember || orgs[1].Host != cloud {
		t.Errorf("orgs = %+v", orgs)
	}
	if *lookups != 1 {
		t.Errorf("credential looked up %d times across 2 pages, want 1", *lookups)
	}
}

func TestListRepos_PaginatesAndMapsFields(t *testing.T) {
	var a *Adapter
	a, _ = newTestAdapter(t, map[string]http.HandlerFunc{
		"/2.0/repositories/team": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") == "2" {
				fmt.Fprint(w, `{"values":[{"slug":"fork","is_private":false,"updated_on":"2026-01-02T03:04:05Z","parent":{"slug":"x"}}]}`)
				return
			}
			fmt.Fprintf(w, `{"values":[{"slug":"api","is_private":true,"updated_on":"2026-09-01T00:00:00Z","parent":null}],"next":%q}`,
				a.baseURL+"/repositories/team?page=2")
		},
	})

	repos, err := a.ListRepos(context.Background(), forge.Org{Name: "team", Host: cloud})
	if err != nil {
		t.Fatal(err)
	}
	want := []forge.Repo{
		{Name: "api", Org: "team", Host: cloud, PushedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Visibility: forge.VisibilityPrivate},
		{Name: "fork", Org: "team", Host: cloud, PushedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Fork: true, Visibility: forge.VisibilityPublic},
	}
	if len(repos) != 2 {
		t.Fatalf("repos = %+v", repos)
	}
	for i := range want {
		if !repos[i].PushedAt.Equal(want[i].PushedAt) {
			t.Errorf("repos[%d].PushedAt = %v, want %v", i, repos[i].PushedAt, want[i].PushedAt)
		}
		repos[i].PushedAt, want[i].PushedAt = time.Time{}, time.Time{}
		if repos[i] != want[i] {
			t.Errorf("repos[%d] = %+v, want %+v", i, repos[i], want[i])
		}
	}
}

func TestCloneURL(t *testing.T) {
	repo := forge.Repo{Name: "api", Org: "team", Host: cloud}
	if got := New().CloneURL(repo); got != "https://bitbucket.org/team/api.git" {
		t.Errorf("https CloneURL = %q", got)
	}
	repo.Host.Protocol = "ssh"
	if got := New().CloneURL(repo); got != "git@bitbucket.org:team/api.git" {
		t.Errorf("ssh CloneURL = %q", got)
	}
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		status    int
		header    string
		wantKind  forge.ErrorKind
		wantRetry time.Duration
	}{
		{http.StatusUnauthorized, "", forge.ErrKindFatal, 0},
		{http.StatusForbidden, "", forge.ErrKindPaneScoped, 0},
		{http.StatusTooManyRequests, "30", forge.ErrKindTransient, 30 * time.Second},
		{http.StatusInternalServerError, "", forge.ErrKindPaneScoped, 0},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			a, lookups := newTestAdapter(t, map[string]http.HandlerFunc{
				"/2.0/repositories/team": func(w http.ResponseWriter, r *http.Request) {
					if tt.header != "" {
						w.Header().Set("Retry-After", tt.header)
					}
					w.WriteHeader(tt.status)
				},
			})
			org := forge.Org{Name: "team", Host: cloud}
			_, err := a.ListRepos(context.Background(), org)

			var fErr *forge.Error
			if !errors.As(err, &fErr) || fErr.Kind != tt.wantKind || fErr.RetryAfter != tt.wantRetry {
				t.Fatalf("error = %#v, want kind %v retry %v", err, tt.wantKind, tt.wantRetry)
			}
			wantFatalWithoutSecret(t, &forge.Error{Kind: forge.ErrKindFatal, Message: fErr.Message})

			// A rejected credential is dropped, so the next call looks it up again.
			a.ListRepos(context.Background(), org)
			wantLookups := 1
			if tt.status == http.StatusUnauthorized {
				wantLookups = 2
			}
			if *lookups != wantLookups {
				t.Errorf("credential lookups = %d, want %d", *lookups, wantLookups)
			}
		})
	}
}

func TestListRepos_RefusesANextLinkToAnotherHost(t *testing.T) {
	a, _ := newTestAdapter(t, map[string]http.HandlerFunc{
		"/2.0/repositories/team": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"values":[{"slug":"api"}],"next":"https://evil.example/steal"}`)
		},
	})
	_, err := a.ListRepos(context.Background(), forge.Org{Name: "team", Host: cloud})
	var fErr *forge.Error
	if !errors.As(err, &fErr) || fErr.Kind != forge.ErrKindPaneScoped {
		t.Errorf("error = %v, want a PaneScoped refusal", err)
	}
}

var dc = forge.Host{Name: "bitbucket.corp.internal", Forge: "bitbucket", Kind: forge.HostPrivate, Protocol: "ssh"}

func TestListOrgs_DataCenterStreamsProjects(t *testing.T) {
	a, lookups := newTestAdapter(t, map[string]http.HandlerFunc{
		"/rest/api/1.0/projects": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("start") == "1" {
				fmt.Fprint(w, `{"values":[{"key":"OPS"}],"isLastPage":true}`)
				return
			}
			fmt.Fprint(w, `{"values":[{"key":"PRJ"}],"isLastPage":false,"nextPageStart":1}`)
		},
	})
	ch, err := a.ListOrgs(context.Background(), dc)
	if err != nil {
		t.Fatal(err)
	}
	var orgs []forge.Org
	for p := range ch {
		if p.Err != nil {
			t.Fatal(p.Err)
		}
		orgs = append(orgs, p.Orgs...)
	}
	want := []forge.Org{{Name: "PRJ", Host: dc}, {Name: "OPS", Host: dc}}
	if len(orgs) != 2 || orgs[0] != want[0] || orgs[1] != want[1] {
		t.Errorf("orgs = %+v, want %+v", orgs, want)
	}
	if *lookups != 1 {
		t.Errorf("credential looked up %d times, want 1", *lookups)
	}
}

func TestListRepos_DataCenterMapsFieldsAndCloneLinks(t *testing.T) {
	a, _ := newTestAdapter(t, map[string]http.HandlerFunc{
		"/rest/api/1.0/projects/PRJ/repos": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("start") == "1" {
				fmt.Fprint(w, `{"values":[{"slug":"web","public":true,"archived":true,"origin":{"slug":"up"},
					"links":{"clone":[{"name":"http","href":"https://me@bitbucket.corp.internal/scm/prj/web.git"}]}}],"isLastPage":true}`)
				return
			}
			fmt.Fprint(w, `{"values":[{"slug":"api","public":false,
				"links":{"clone":[{"name":"http","href":"https://me@bitbucket.corp.internal/scm/prj/api.git"},
				{"name":"ssh","href":"ssh://git@bitbucket.corp.internal:7999/prj/api.git"}]}}],"isLastPage":false,"nextPageStart":1}`)
		},
	})
	repos, err := a.ListRepos(context.Background(), forge.Org{Name: "PRJ", Host: dc})
	if err != nil {
		t.Fatal(err)
	}
	want := []forge.Repo{
		{Name: "api", Org: "PRJ", Host: dc, Visibility: forge.VisibilityPrivate, CloneLink: "ssh://git@bitbucket.corp.internal:7999/prj/api.git"},
		// No ssh link listed: the http link is used, without the embedded user.
		{Name: "web", Org: "PRJ", Host: dc, Archived: true, Fork: true, Visibility: forge.VisibilityPublic,
			CloneLink: "https://bitbucket.corp.internal/scm/prj/web.git"},
	}
	if len(repos) != 2 || repos[0] != want[0] || repos[1] != want[1] {
		t.Fatalf("repos = %+v\nwant %+v", repos, want)
	}
	if got := a.CloneURL(repos[0]); got != want[0].CloneLink {
		t.Errorf("CloneURL = %q", got)
	}

	https := dc
	https.Protocol = "https"
	repos, _ = a.ListRepos(context.Background(), forge.Org{Name: "PRJ", Host: https})
	if len(repos) == 0 || repos[0].CloneLink != "https://bitbucket.corp.internal/scm/prj/api.git" {
		t.Errorf("https repos = %+v", repos)
	}
}

func TestDataCenterAndCloudKeepSeparateCredentials(t *testing.T) {
	a, lookups := newTestAdapter(t, map[string]http.HandlerFunc{
		"/rest/api/1.0/projects": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"values":[],"isLastPage":true}`)
		},
		"/2.0/user/workspaces": func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"values":[]}`) },
	})
	var hosts []string
	fill := a.fill
	a.fill = func(ctx context.Context, h string) (credential, bool, error) {
		hosts = append(hosts, h)
		return fill(ctx, h)
	}
	for _, h := range []forge.Host{dc, cloud, dc} {
		if _, err := a.ListOrgs(context.Background(), h); err != nil {
			t.Fatal(err)
		}
	}
	if *lookups != 2 || hosts[0] != "bitbucket.corp.internal" {
		t.Errorf("lookups = %d for hosts %v, want one per Host, Data Center asking for its own host", *lookups, hosts)
	}
}

func TestDataCenterBearerTokenFromEnvironment(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		fmt.Fprint(w, `{"values":[],"isLastPage":true}`)
	}))
	t.Cleanup(srv.Close)
	a := &Adapter{
		http:   srv.Client(),
		dcBase: func(string) string { return srv.URL },
		getenv: env(map[string]string{"BITBUCKET_TOKEN": secret, "BITBUCKET_EMAIL": "cloud@example.com", "BITBUCKET_API_TOKEN": "cloud"}),
		fill:   fillMustNotRun(t),
	}
	if _, err := a.ListOrgs(context.Background(), dc); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer "+secret {
		t.Errorf("Authorization = %q, want the Bearer token", auth)
	}
}
