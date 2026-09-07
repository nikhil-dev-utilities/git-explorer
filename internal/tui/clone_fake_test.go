package tui

import (
	"context"
	"sync"

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

// noopCloneRunner is used by tests that don't exercise a Clone Run at all — it
// should never be called.
func noopCloneRunner(context.Context, string, []clone.Repo, bool, int) []clone.Result {
	return nil
}

// fakeCloneRunner is the CloneRunnerFunc every Clone-Run cancellation test in this
// package injects. It blocks on ctx.Done(), simulating an in-flight run, so a test
// can send a cancel key and observe the transition — this package's tests never
// touch git. Safe for concurrent use: bubbletea runs it on its own goroutine while
// the test polls snapshotCallCount from another.
type fakeCloneRunner struct {
	// byRepo, if set, overrides the default Cloned outcome for the named repo.
	byRepo map[string]clone.Result

	mu    sync.Mutex
	calls []fakeCloneRunnerCall
}

type fakeCloneRunnerCall struct {
	target      string
	repos       []clone.Repo
	orgSubdir   bool
	parallelism int
}

func (f *fakeCloneRunner) fn() CloneRunnerFunc {
	return func(ctx context.Context, target string, repos []clone.Repo, orgSubdir bool, parallelism int) []clone.Result {
		f.mu.Lock()
		f.calls = append(f.calls, fakeCloneRunnerCall{target: target, repos: repos, orgSubdir: orgSubdir, parallelism: parallelism})
		f.mu.Unlock()

		<-ctx.Done() // block until the test cancels, simulating an in-flight run

		out := make([]clone.Result, len(repos))
		for i, r := range repos {
			if res, ok := f.byRepo[r.Name]; ok {
				res.Repo = r
				out[i] = res
				continue
			}
			out[i] = clone.Result{Repo: r, Dest: clone.TargetPath(target, r, orgSubdir), Outcome: clone.OutcomeCloned}
		}
		return out
	}
}

func (f *fakeCloneRunner) snapshotCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// fakeCloneRunnerImmediate is a CloneRunnerFunc that returns immediately (no
// cancellation wait) — for tests that just want a completed run without exercising
// the cancel path.
func fakeCloneRunnerImmediate(byRepo map[string]clone.Result) CloneRunnerFunc {
	return func(_ context.Context, target string, repos []clone.Repo, orgSubdir bool, _ int) []clone.Result {
		out := make([]clone.Result, len(repos))
		for i, r := range repos {
			if res, ok := byRepo[r.Name]; ok {
				res.Repo = r
				out[i] = res
				continue
			}
			out[i] = clone.Result{Repo: r, Dest: clone.TargetPath(target, r, orgSubdir), Outcome: clone.OutcomeCloned}
		}
		return out
	}
}
