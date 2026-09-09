package clone

import (
	"context"
	"log/slog"
	"sync"
)

// Result is one Repo's outcome from a Run.
type Result struct {
	Repo    Repo
	Dest    string
	Outcome Outcome
	// Err is non-nil if classification failed, or — for a Cloned Outcome — if the
	// clone itself failed. Skipped and Conflict never reach an execution step, so
	// they never carry an Err from one. Check Err before trusting Outcome.
	Err error
}

// Run classifies every Repo in repos against target, then clones whichever classify
// as Cloned — bounded to at most parallelism concurrent git clone subprocesses at any
// time, never exceeded even transiently. It never aborts on a single failure: every
// Repo gets a Result, regardless of what happened to any other, and Skipped/Conflict
// Repos are reported with no clone subprocess ever attempted for them.
//
// Cancelling ctx stops any further clone from being dispatched — checked explicitly
// before each one, not left to chance — and terminates whichever clone(s) are already
// in flight (runGit's exec.CommandContext kills the subprocess when ctx is done).
// Repos that had already completed before cancellation keep their Outcome and Result
// untouched; Repos never dispatched get ctx.Err() as their Result's Err. Run returns
// as soon as every already-dispatched clone has actually finished (successfully,
// with an error, or killed by the cancellation) — it does not wait on work that was
// never started.
func Run(ctx context.Context, target string, repos []Repo, orgSubdir bool, parallelism int) []Result {
	slog.InfoContext(ctx, "clone run starting", "target", target, "repos", len(repos), "parallelism", parallelism, "org_subdir", orgSubdir)

	results := make([]Result, len(repos))

	type job struct {
		index int
		repo  Repo
		dest  string
	}
	var jobs []job

	for i, repo := range repos {
		outcome, dest, err := Classify(ctx, target, repo, orgSubdir)
		results[i] = Result{Repo: repo, Dest: dest, Outcome: outcome, Err: err}
		if err == nil && outcome == OutcomeCloned {
			jobs = append(jobs, job{index: i, repo: repo, dest: dest})
			continue
		}
		// Skipped/Conflict (or a classification error) never reach the goroutine
		// below, so they get their progress line here instead — every Repo gets one
		// as soon as its outcome is known, not just the ones that hit git.
		logRepoOutcome(ctx, results[i])
	}

	if parallelism < 1 {
		parallelism = 1
	}
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup

dispatch:
	for _, j := range jobs {
		if ctx.Err() != nil {
			results[j.index].Err = ctx.Err()
			continue
		}
		select {
		case <-ctx.Done():
			results[j.index].Err = ctx.Err()
			continue dispatch
		case sem <- struct{}{}:
			// select's case order is not a preference: if ctx became Done at
			// essentially the same moment a slot freed, both cases can be ready
			// simultaneously and select picks between them at random. Re-check
			// deterministically rather than let that coin flip decide whether one
			// more clone gets dispatched after cancellation.
			if ctx.Err() != nil {
				<-sem
				results[j.index].Err = ctx.Err()
				continue dispatch
			}
		}

		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()
			results[j.index].Err = performClone(ctx, j.repo.CloneURL, j.dest)
			logRepoOutcome(ctx, results[j.index])
		}(j)
	}
	wg.Wait()

	logRunSummary(ctx, results)
	return results
}

// logRepoOutcome is the "progress per repo" line: logged the moment each Repo's
// outcome is actually known — synchronously in the classification loop for
// Skipped/Conflict, from inside the goroutine right after performClone returns for
// everything else — rather than waiting for the whole batch to finish. This is what
// makes tailing the log file during a long run show live progress instead of
// silence until one final summary.
func logRepoOutcome(ctx context.Context, r Result) {
	if r.Err != nil {
		slog.WarnContext(ctx, "clone failed", "repo", r.Repo.Name, "org", r.Repo.Org, "dest", r.Dest, "error", r.Err)
		return
	}
	slog.InfoContext(ctx, "clone progress", "repo", r.Repo.Name, "org", r.Repo.Org, "dest", r.Dest, "outcome", r.Outcome)
}

// logRunSummary counts Results by Outcome (Err takes precedence over whatever
// Outcome a Repo classified as, matching Result.Err's own documented precedence) and
// logs the aggregate at info. Individual outcomes are already logged live as they
// happen (logRepoOutcome) — this is a summary on top, not a second copy of every line.
func logRunSummary(ctx context.Context, results []Result) {
	var cloned, skipped, conflict, failed int
	for _, r := range results {
		switch {
		case r.Err != nil:
			failed++
		case r.Outcome == OutcomeSkipped:
			skipped++
		case r.Outcome == OutcomeConflict:
			conflict++
		default:
			cloned++
		}
	}
	slog.InfoContext(ctx, "clone run finished", "cloned", cloned, "skipped", skipped, "conflict", conflict, "failed", failed)
}
