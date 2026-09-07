package tui

import (
	"context"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

// fakeClonePreview is the ClonePreviewFunc every test in this package injects
// instead of a real one wrapping clone.Classify — this package's tests never touch
// the filesystem or git.
type fakeClonePreview struct {
	// results, if set, is returned verbatim regardless of input.
	results []clone.Result
	// byRepo, if set (and results is nil), builds a Cloned result for every repo,
	// unless an override is present here.
	byRepo map[string]clone.Result

	calls []fakeClonePreviewCall
}

type fakeClonePreviewCall struct {
	target    string
	repos     []clone.Repo
	orgSubdir bool
}

func (f *fakeClonePreview) fn() ClonePreviewFunc {
	return func(_ context.Context, target string, repos []clone.Repo, orgSubdir bool) []clone.Result {
		f.calls = append(f.calls, fakeClonePreviewCall{target: target, repos: repos, orgSubdir: orgSubdir})
		if f.results != nil {
			return f.results
		}
		out := make([]clone.Result, len(repos))
		for i, r := range repos {
			if res, ok := f.byRepo[r.Name]; ok {
				res.Repo = r // byRepo entries only specify Outcome/Dest; identity comes from the call
				out[i] = res
				continue
			}
			out[i] = clone.Result{
				Repo:    r,
				Dest:    clone.TargetPath("/src", r, orgSubdir),
				Outcome: clone.OutcomeCloned,
			}
		}
		return out
	}
}

// noopClonePreview is used by tests that don't exercise the clone dialog at all —
// it should never be called.
func noopClonePreview(context.Context, string, []clone.Repo, bool) []clone.Result {
	return nil
}
