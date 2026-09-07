package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
)

// newBareRepo creates a fresh local bare repository, usable as a clone source, per
// ADR-0003's testing decision (mirroring internal/clone's own test helper): exercise
// the real git binary against real local repositories, never a mock.
func newBareRepo(t *testing.T) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "repo.git")
	cmd := exec.Command("git", "init", "--bare", "-q", bare)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init --bare: %v", err)
	}
	return bare
}

func TestPreviewClones_ClassifiesEachRepoWithoutCloning(t *testing.T) {
	freshBare := newBareRepo(t)
	target := t.TempDir()

	repos := []clone.Repo{
		{Org: "acme", Name: "api", CloneURL: freshBare},
		{Org: "acme", Name: "web", CloneURL: freshBare},
	}

	results := previewClones(context.Background(), target, repos, false)

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
		}
		if r.Outcome != clone.OutcomeCloned {
			t.Errorf("results[%d].Outcome = %v, want Cloned (path absent)", i, r.Outcome)
		}
		wantDest := filepath.Join(target, repos[i].Name)
		if r.Dest != wantDest {
			t.Errorf("results[%d].Dest = %q, want %q", i, r.Dest, wantDest)
		}
	}

	// previewClones must never actually clone — the whole point is a side-effect-free
	// pre-flight check.
	entries, err := filepath.Glob(filepath.Join(target, "*"))
	if err != nil {
		t.Fatalf("glob target: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("target directory has entries after previewClones: %v, want none", entries)
	}
}

func TestPreviewClones_OrgSubdirChangesDestination(t *testing.T) {
	bare := newBareRepo(t)
	target := t.TempDir()
	repos := []clone.Repo{{Org: "acme", Name: "api", CloneURL: bare}}

	results := previewClones(context.Background(), target, repos, true)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	wantDest := filepath.Join(target, "acme", "api")
	if results[0].Dest != wantDest {
		t.Errorf("Dest = %q, want %q", results[0].Dest, wantDest)
	}
}

func TestPreviewClones_EmptyReposReturnsEmptyResults(t *testing.T) {
	results := previewClones(context.Background(), t.TempDir(), nil, false)
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}
