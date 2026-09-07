package main

import (
	"context"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

// previewClones is the batched tui.ClonePreviewFunc adapter — a thin loop over
// clone.Classify. tui.ClonePreviewFunc's own doc comment names this exact shape as
// composition-root wiring, not part of internal/clone or internal/tui: clone.Classify
// is per-Repo, and neither package needed to change to support batching it here.
func previewClones(ctx context.Context, target string, repos []clone.Repo, orgSubdir bool) []clone.Result {
	results := make([]clone.Result, len(repos))
	for i, r := range repos {
		outcome, dest, err := clone.Classify(ctx, target, r, orgSubdir)
		results[i] = clone.Result{Repo: r, Dest: dest, Outcome: outcome, Err: err}
	}
	return results
}
