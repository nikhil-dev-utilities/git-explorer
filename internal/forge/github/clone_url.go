package github

import (
	"fmt"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// CloneURL builds the clone URL for repo, respecting its Host's configured Protocol.
// Any Protocol value other than "https" is treated as ssh — ssh is the documented
// default (DESIGN.md), so an unset or unrecognized Protocol degrades to it rather than
// to an unusable URL.
func (a *Adapter) CloneURL(repo forge.Repo) string {
	if repo.Host.Protocol == "https" {
		return fmt.Sprintf("https://%s/%s/%s.git", repo.Host.Name, repo.Org, repo.Name)
	}
	return fmt.Sprintf("git@%s:%s/%s.git", repo.Host.Name, repo.Org, repo.Name)
}
