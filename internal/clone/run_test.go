package clone

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRun_MixedOutcomes(t *testing.T) {
	target := t.TempDir()

	freshBare := newBareRepo(t)
	freshRepo := Repo{Org: "acme", Name: "fresh", CloneURL: freshBare}

	skippedRepo := Repo{Org: "acme", Name: "skipped", CloneURL: "https://example.invalid/acme/skipped.git"}
	initRepoWithOrigin(t, TargetPath(target, skippedRepo, false), "https://example.invalid/acme/skipped.git")

	conflictRepo := Repo{Org: "acme", Name: "conflict", CloneURL: "https://example.invalid/acme/conflict.git"}
	conflictDest := TargetPath(target, conflictRepo, false)
	if err := os.MkdirAll(conflictDest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conflictDest, "occupied.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	results := Run(context.Background(), target, []Repo{freshRepo, skippedRepo, conflictRepo}, false, 8)
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	byName := make(map[string]Result, 3)
	for _, r := range results {
		byName[r.Repo.Name] = r
	}

	if got := byName["fresh"]; got.Outcome != OutcomeCloned || got.Err != nil {
		t.Errorf("fresh = %+v, want Cloned with no error", got)
	}
	if _, err := os.Stat(filepath.Join(byName["fresh"].Dest, ".git")); err != nil {
		t.Errorf("fresh repo not actually cloned to disk: %v", err)
	}

	if got := byName["skipped"]; got.Outcome != OutcomeSkipped || got.Err != nil {
		t.Errorf("skipped = %+v, want Skipped with no error", got)
	}

	if got := byName["conflict"]; got.Outcome != OutcomeConflict || got.Err != nil {
		t.Errorf("conflict = %+v, want Conflict with no error", got)
	}
	data, err := os.ReadFile(filepath.Join(conflictDest, "occupied.txt"))
	if err != nil || string(data) != "x" {
		t.Errorf("conflict path was modified: data=%q err=%v", data, err)
	}
}

func TestRun_OneFailureDoesNotStopTheRest(t *testing.T) {
	target := t.TempDir()

	goodRepo1 := Repo{Org: "acme", Name: "good1", CloneURL: newBareRepo(t)}
	goodRepo2 := Repo{Org: "acme", Name: "good2", CloneURL: newBareRepo(t)}
	badRepo := Repo{Org: "acme", Name: "bad", CloneURL: filepath.Join(t.TempDir(), "does-not-exist.git")}

	results := Run(context.Background(), target, []Repo{goodRepo1, badRepo, goodRepo2}, false, 4)

	byName := make(map[string]Result, 3)
	for _, r := range results {
		byName[r.Repo.Name] = r
	}

	if err := byName["good1"].Err; err != nil {
		t.Errorf("good1.Err = %v, want nil (one bad repo must not affect the others)", err)
	}
	if err := byName["good2"].Err; err != nil {
		t.Errorf("good2.Err = %v, want nil", err)
	}
	if byName["bad"].Err == nil {
		t.Error("bad.Err = nil, want an error reported for the unreachable source")
	}

	for _, name := range []string{"good1", "good2"} {
		if _, err := os.Stat(filepath.Join(byName[name].Dest, ".git")); err != nil {
			t.Errorf("%s not actually cloned to disk: %v", name, err)
		}
	}
}

func TestRun_ParallelismBoundNeverExceeded(t *testing.T) {
	const parallelism = 3
	const numRepos = 15

	target := t.TempDir()
	repos := make([]Repo, numRepos)
	for i := range repos {
		repos[i] = Repo{Org: "acme", Name: fmt.Sprintf("repo-%d", i), CloneURL: newBareRepo(t)}
	}

	var maxObserved int64
	stop := make(chan struct{})
	var pollWG sync.WaitGroup
	pollWG.Add(1)
	go func() {
		defer pollWG.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if v := atomic.LoadInt64(&activeClones); v > atomic.LoadInt64(&maxObserved) {
				atomic.StoreInt64(&maxObserved, v)
			}
			time.Sleep(50 * time.Microsecond)
		}
	}()

	results := Run(context.Background(), target, repos, false, parallelism)

	close(stop)
	pollWG.Wait()

	for _, r := range results {
		if r.Err != nil {
			t.Errorf("repo %s: unexpected error %v", r.Repo.Name, r.Err)
		}
	}
	got := atomic.LoadInt64(&maxObserved)
	t.Logf("max concurrent clones observed: %d (bound: %d)", got, parallelism)
	if got > int64(parallelism) {
		t.Errorf("observed %d concurrent clones, want never more than %d", got, parallelism)
	}
	if got == 0 {
		t.Error("observed 0 concurrent clones — the poller likely isn't sampling during real work, this test isn't verifying anything")
	}
}
