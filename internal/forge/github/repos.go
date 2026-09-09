package github

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// ListRepos is called lazily — only once an Org has been selected — and returns each
// Repo with the fields the Repo pane needs to render and filter: name, PushedAt,
// Archived, Fork, and Visibility.
func (a *Adapter) ListRepos(ctx context.Context, org forge.Org) ([]forge.Repo, error) {
	entries, err := fetchAllPages[repoEntry](ctx, a, org.Host, "orgs/"+org.Name+"/repos")
	if err != nil {
		return nil, err
	}

	repos := make([]forge.Repo, len(entries))
	for i, e := range entries {
		var pushedAt time.Time
		if e.PushedAt != nil {
			pushedAt = *e.PushedAt
		}
		repos[i] = forge.Repo{
			Name:       e.Name,
			Org:        org.Name,
			Host:       org.Host,
			PushedAt:   pushedAt,
			Archived:   e.Archived,
			Fork:       e.Fork,
			Visibility: parseVisibility(e.Visibility),
		}
	}
	slog.InfoContext(ctx, "listed repos", "org", org.Name, "host", org.Host.Name, "count", len(repos))
	return repos, nil
}

type repoEntry struct {
	Name string `json:"name"`
	// PushedAt is nullable in GitHub's API — a repo with no commits ever pushed to it
	// has it as null. A nil pointer here becomes forge.Repo's zero time.Time.
	PushedAt   *time.Time `json:"pushed_at"`
	Archived   bool       `json:"archived"`
	Fork       bool       `json:"fork"`
	Visibility string     `json:"visibility"`
}

func parseVisibility(s string) forge.Visibility {
	switch s {
	case "private":
		return forge.VisibilityPrivate
	case "internal":
		return forge.VisibilityInternal
	default:
		return forge.VisibilityPublic
	}
}
