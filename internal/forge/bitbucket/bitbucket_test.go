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

func TestListOrgs_DataCenterHostIsFatal(t *testing.T) {
	_, err := New().ListOrgs(context.Background(), forge.Host{Name: "bitbucket.corp.internal", Forge: "bitbucket"})
	var fErr *forge.Error
	if !errors.As(err, &fErr) || fErr.Kind != forge.ErrKindFatal {
		t.Errorf("error = %v, want Fatal", err)
	}
}
