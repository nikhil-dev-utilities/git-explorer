package bitbucket

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// Bitbucket Data Center maps Project to Org (ADR-0010, ADR-0012). Its REST API lives on
// the Host itself under /rest/api/1.0 and pages with start, limit and isLastPage.

// dcPage is Data Center's pagination envelope.
type dcPage[T any] struct {
	Values        []T  `json:"values"`
	IsLastPage    bool `json:"isLastPage"`
	NextPageStart int  `json:"nextPageStart"`
}

const dcPageLimit = 100

// getDCPage fetches the page of path that begins at start. last is true on the final
// page, or when the server's next start would not move forward.
func getDCPage[T any](ctx context.Context, a *Adapter, host forge.Host, path string, start int) (values []T, next int, last bool, err error) {
	var p dcPage[T]
	u := fmt.Sprintf("%s%s?limit=%d&start=%d", a.dcBase(host.Name), path, dcPageLimit, start)
	if err := a.get(ctx, host, u, &p); err != nil {
		return nil, 0, true, err
	}
	return p.Values, p.NextPageStart, p.IsLastPage || p.NextPageStart <= start, nil
}

type dcProject struct {
	Key string `json:"key"`
}

// listProjects lists every Project on the Host as an Org. A Private Host lists every
// Visible Org (ADR-0002). Data Center has no cheap per-Project membership call, so
// Affiliation is None. The first page is fetched synchronously, so a missing or
// rejected credential is a synchronous Fatal; later pages stream as they arrive.
func (a *Adapter) listProjects(ctx context.Context, host forge.Host) (<-chan forge.OrgPage, error) {
	slog.InfoContext(ctx, "listing orgs", "host", host.Name)
	projects, next, last, err := getDCPage[dcProject](ctx, a, host, "/projects", 0)
	if err != nil {
		return nil, err
	}
	toOrgs := func(ps []dcProject) []forge.Org {
		orgs := make([]forge.Org, len(ps))
		for i, p := range ps {
			orgs[i] = forge.Org{Name: p.Key, Host: host}
		}
		return orgs
	}

	out := make(chan forge.OrgPage, 1)
	out <- forge.OrgPage{Orgs: toOrgs(projects)}
	go func() {
		defer close(out)
		for !last {
			start := next
			projects, next, last, err = getDCPage[dcProject](ctx, a, host, "/projects", start)
			page := forge.OrgPage{Orgs: toOrgs(projects), Err: err}
			select {
			case out <- page:
			case <-ctx.Done():
				return
			}
			slog.InfoContext(ctx, "fetched orgs page", "host", host.Name, "start", start, "count", len(projects))
		}
	}()
	return out, nil
}

type dcRepo struct {
	Slug     string    `json:"slug"`
	Public   bool      `json:"public"`
	Archived bool      `json:"archived"`
	Origin   *struct{} `json:"origin"`
	Links    struct {
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
	} `json:"links"`
}

// listProjectRepos lists every Repo in the Project. Data Center reports no push time,
// so PushedAt is zero.
func (a *Adapter) listProjectRepos(ctx context.Context, org forge.Org) ([]forge.Repo, error) {
	path := "/projects/" + url.PathEscape(org.Name) + "/repos"
	var repos []forge.Repo
	for start, last := 0, false; !last; {
		var entries []dcRepo
		var err error
		entries, start, last, err = getDCPage[dcRepo](ctx, a, org.Host, path, start)
		if err != nil {
			return repos, err
		}
		for _, e := range entries {
			vis := forge.VisibilityPrivate
			if e.Public {
				vis = forge.VisibilityPublic
			}
			repos = append(repos, forge.Repo{
				Name:       e.Slug,
				Org:        org.Name,
				Host:       org.Host,
				Archived:   e.Archived,
				Fork:       e.Origin != nil,
				Visibility: vis,
				CloneLink:  e.cloneLink(org.Host.Protocol),
			})
		}
	}
	slog.InfoContext(ctx, "listed repos", "org", org.Name, "host", org.Host.Name, "count", len(repos))
	return repos, nil
}

// cloneLink picks the listed clone URL for protocol: "https" takes the http link,
// anything else the ssh link, as for Cloud. The ssh port and context path differ per
// instance, which is why the link is used rather than built. The http link embeds the
// API user; it is dropped so git asks its credential helper as usual. If the wanted
// link is missing (ssh disabled on the instance), the other one is used.
func (e dcRepo) cloneLink(protocol string) string {
	want := "ssh"
	if protocol == "https" {
		want = "http"
	}
	var link string
	for _, l := range e.Links.Clone {
		if l.Name == want || link == "" {
			link = l.Href
		}
	}
	if u, err := url.Parse(link); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		u.User = nil
		return u.String()
	}
	return link
}
