package bitbucket

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

var _ forge.Forge = (*Adapter)(nil)

// cloudHost is Bitbucket Cloud. Every other Bitbucket Host is Data Center (ADR-0012).
const cloudHost = "bitbucket.org"

func isDataCenter(host forge.Host) bool { return host.Name != cloudHost }

// Adapter implements forge.Forge for Bitbucket Cloud and Data Center. Construct it with New.
type Adapter struct {
	http    *http.Client
	baseURL string                   // Cloud REST API root, e.g. https://api.bitbucket.org/2.0
	dcBase  func(host string) string // Data Center REST API root for a host
	getenv  func(string) string
	fill    credentialFillFunc

	mu    sync.Mutex
	creds map[string]credential // by Host name, resolved once per session, dropped on a 401
}

// New returns an Adapter that talks to the real Bitbucket APIs.
func New() *Adapter {
	return &Adapter{
		http:    &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://api.bitbucket.org/2.0",
		dcBase:  func(host string) string { return "https://" + host + "/rest/api/1.0" },
		getenv:  os.Getenv,
		fill:    gitCredentialFill,
	}
}

// ListOrgs lists the caller's Workspaces as Orgs. Bitbucket only lists Workspaces the
// caller can access, which is the Public Host rule of ADR-0002. The list is small, so
// it is fetched synchronously and delivered as one page.
func (a *Adapter) ListOrgs(ctx context.Context, host forge.Host) (<-chan forge.OrgPage, error) {
	if isDataCenter(host) {
		return a.listProjects(ctx, host)
	}
	slog.InfoContext(ctx, "listing orgs", "host", host.Name)

	entries, err := fetchAll[workspaceAccess](ctx, a, host, a.baseURL+"/user/workspaces?pagelen=100")
	if err != nil {
		return nil, err
	}
	orgs := make([]forge.Org, len(entries))
	for i, e := range entries {
		aff := forge.AffiliationMember
		if e.Administrator {
			aff = forge.AffiliationOwner
		}
		orgs[i] = forge.Org{Name: e.Workspace.Slug, Host: host, Affiliation: aff}
	}
	slog.InfoContext(ctx, "listed orgs", "host", host.Name, "count", len(orgs))

	page := make(chan forge.OrgPage, 1)
	page <- forge.OrgPage{Orgs: orgs}
	close(page)
	return page, nil
}

// workspaceAccess is one entry of GET /user/workspaces. It has no member/collaborator
// distinction, only administrator, so Affiliation is Owner or Member.
type workspaceAccess struct {
	Administrator bool `json:"administrator"`
	Workspace     struct {
		Slug string `json:"slug"`
	} `json:"workspace"`
}

// ListRepos lists every Repo in the Workspace the credential can see. Projects are
// ignored (ADR-0010). Bitbucket has no archived flag, so Archived is always false.
func (a *Adapter) ListRepos(ctx context.Context, org forge.Org) ([]forge.Repo, error) {
	if isDataCenter(org.Host) {
		return a.listProjectRepos(ctx, org)
	}
	entries, err := fetchAll[repoEntry](ctx, a, org.Host, a.baseURL+"/repositories/"+url.PathEscape(org.Name)+"?pagelen=100")
	if err != nil {
		return nil, err
	}
	repos := make([]forge.Repo, len(entries))
	for i, e := range entries {
		vis := forge.VisibilityPublic
		if e.IsPrivate {
			vis = forge.VisibilityPrivate
		}
		repos[i] = forge.Repo{
			Name:       e.Slug,
			Org:        org.Name,
			Host:       org.Host,
			PushedAt:   e.UpdatedOn,
			Fork:       e.Parent != nil,
			Visibility: vis,
		}
	}
	slog.InfoContext(ctx, "listed repos", "org", org.Name, "host", org.Host.Name, "count", len(repos))
	return repos, nil
}

type repoEntry struct {
	Slug      string    `json:"slug"`
	IsPrivate bool      `json:"is_private"`
	UpdatedOn time.Time `json:"updated_on"`
	Parent    *struct{} `json:"parent"`
}

// CloneURL builds the clone URL from the Workspace and Repo slugs. Bitbucket's own
// links.clone https entry embeds the API user (an email), which is the wrong username
// for git (ADR-0011), so the URL is built here instead. Any Protocol other than
// "https" is treated as ssh, as for GitHub. On Data Center the link listed with the
// Repo is used as is.
func (a *Adapter) CloneURL(repo forge.Repo) string {
	if isDataCenter(repo.Host) {
		return repo.CloneLink
	}
	if repo.Host.Protocol == "https" {
		return fmt.Sprintf("https://%s/%s/%s.git", repo.Host.Name, repo.Org, repo.Name)
	}
	return fmt.Sprintf("git@%s:%s/%s.git", repo.Host.Name, repo.Org, repo.Name)
}
