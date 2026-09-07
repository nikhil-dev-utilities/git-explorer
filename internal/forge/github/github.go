// Package github implements forge.Forge for GitHub (github.com and self-managed
// enterprise installs), reached through the gh CLI Frontdoor. See ADR-0001.
package github

import (
	"context"
	"fmt"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

var _ forge.Forge = (*Adapter)(nil)

// Adapter implements forge.Forge. Construct it with New; newWithRunner exists for this
// package's own tests to inject a fake runner.
type Adapter struct {
	run runner
}

// New returns an Adapter backed by the real gh binary on PATH.
func New() *Adapter {
	return &Adapter{run: execRunner{}}
}

func newWithRunner(r runner) *Adapter {
	return &Adapter{run: r}
}

// ListRepos is implemented in a later slice of this PRD (issue #8).
func (a *Adapter) ListRepos(ctx context.Context, org forge.Org) ([]forge.Repo, error) {
	return nil, fmt.Errorf("github: ListRepos not implemented yet")
}

// CloneURL is implemented in a later slice of this PRD (issue #8). It panics rather
// than silently returning an unusable value, since nothing in this codebase calls it
// yet — a panic here means a caller was wired up ahead of the implementation.
func (a *Adapter) CloneURL(repo forge.Repo) string {
	panic("github: CloneURL not implemented yet")
}
