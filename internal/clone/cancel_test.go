package clone

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// waitForActiveClones polls activeClones (real production instrumentation — see
// clone.go) until it satisfies want, deterministically synchronizing with an actual
// clone subprocess's lifecycle rather than guessing a sleep duration. It returns
// whether want was satisfied before the timeout, rather than calling t.Fatal itself —
// this is called from background goroutines, and testing.T's Fatal/FailNow must only
// be called from the test's own goroutine.
func waitForActiveClones(want func(int64) bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if want(atomic.LoadInt64(&activeClones)) {
			return true
		}
		time.Sleep(100 * time.Microsecond)
	}
	return false
}

func TestRun_CancellationStopsDispatchingNewClones(t *testing.T) {
	target := t.TempDir()
	// "first" is deliberately slow (a real payload, tens to low-hundreds of ms to
	// clone locally) so there is no ambiguity about which job the poller below is
	// observing — it's cancelled while still in flight, well before "second" or
	// "third" (both trivially fast, empty bare repos) could possibly have started:
	// with parallelism=1, dispatch is strictly serial, so neither can begin until
	// "first" releases its semaphore slot, which only happens once it's done —
	// after cancellation has already been observed by the dispatch loop.
	//
	// An earlier version of this test tried to synchronize by waiting for
	// activeClones to cycle 1 -> 0 for a same-speed "first" job, on the assumption
	// that whichever cycle the poller caught would be job 1's. That assumption was
	// wrong: goroutine scheduling gives no guarantee the poller runs before job 1
	// (or even job 2) has already finished, so it could just as easily observe job
	// 2's cycle instead. Making job 1 unambiguously the slow one removes that
	// ambiguity rather than fighting it with a tighter poll interval.
	repos := []Repo{
		{Org: "acme", Name: "first", CloneURL: newBareRepoWithBlob(t, 30)},
		{Org: "acme", Name: "second", CloneURL: newBareRepo(t)},
		{Org: "acme", Name: "third", CloneURL: newBareRepo(t)},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		waitForActiveClones(func(v int64) bool { return v >= 1 }, 10*time.Second)
		cancel()
	}()

	results := Run(ctx, target, repos, false, 1)
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	// "first" is racy by design (cancellation may land before or after it
	// finishes) — not asserted on here; TestRun_CancellationTerminatesAnInFlightClone
	// covers that specific property with an unambiguous single-job test.
	for _, r := range results[1:] {
		if r.Err == nil {
			t.Errorf("%s.Err = nil, want an error (it must never have been dispatched)", r.Repo.Name)
		}
		if _, err := os.Stat(r.Dest); !os.IsNotExist(err) {
			t.Errorf("%s was cloned to disk despite cancellation (stat err = %v)", r.Repo.Name, err)
		}
	}
}

// newBareRepoWithBlob creates a bare repo whose single commit contains an
// incompressible blob of the given size, making a local clone of it take long enough
// to reliably still be in flight when cancelled shortly after it starts.
func newBareRepoWithBlob(t *testing.T, sizeMB int) string {
	t.Helper()
	work := t.TempDir()
	mustRunGitIn(t, work, "init", "-q")
	mustRunGitIn(t, work, "config", "user.email", "test@example.com")
	mustRunGitIn(t, work, "config", "user.name", "test")

	data := make([]byte, sizeMB*1024*1024)
	rand.New(rand.NewSource(1)).Read(data) //nolint:gosec // test fixture, not security-sensitive
	if err := os.WriteFile(filepath.Join(work, "blob.bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	mustRunGitIn(t, work, "add", "blob.bin")
	mustRunGitIn(t, work, "commit", "-q", "-m", "add blob")

	bare := filepath.Join(t.TempDir(), "repo.git")
	mustRunGit(t, "clone", "--bare", "-q", work, bare)
	return bare
}

func TestRun_CancellationTerminatesAnInFlightClone(t *testing.T) {
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "big", CloneURL: newBareRepoWithBlob(t, 40)}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		waitForActiveClones(func(v int64) bool { return v >= 1 }, 10*time.Second)
		cancel()
	}()

	start := time.Now()
	results := Run(ctx, target, []Repo{repo}, false, 1)
	elapsed := time.Since(start)

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Err == nil {
		t.Error("Err = nil, want the in-flight clone to have been terminated rather than completing")
	}
	// A full uncancelled clone of a 40MB blob takes at least tens of milliseconds
	// locally; returning promptly after cancellation should be well under that.
	if elapsed > 5*time.Second {
		t.Errorf("Run() took %v to return after cancellation, want it to return promptly", elapsed)
	}
	t.Logf("Run() returned in %v after cancellation", elapsed)
}
