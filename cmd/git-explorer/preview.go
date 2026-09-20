package main

import (
	"context"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

// previewClones is the batched glyphclone.PreviewFunc adapter — a thin loop over
// clone.Classify, which is per-Repo. It is composition-root wiring, not part of
// internal/clone or internal/glyphclone.
func previewClones(ctx context.Context, target string, repos []clone.Repo, orgSubdir bool) []clone.Result {
	results := make([]clone.Result, len(repos))
	for i, r := range repos {
		outcome, dest, err := clone.Classify(ctx, target, r, orgSubdir)
		results[i] = clone.Result{Repo: r, Dest: dest, Outcome: outcome, Err: err}
	}
	return results
}
