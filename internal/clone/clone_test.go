package clone

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func mustRunGit(t *testing.T, args ...string) {
	t.Helper()
	if _, err := runGit(context.Background(), args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

// newBareRepo creates a fresh local bare repository, usable as a clone source, per
// ADR-0003's testing decision: exercise the real git binary against real local
// repositories, never a mock.
func newBareRepo(t *testing.T) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "repo.git")
	mustRunGit(t, "init", "--bare", "-q", bare)
	return bare
}

func TestTargetPath(t *testing.T) {
	repo := Repo{Org: "acme", Name: "api"}

	if got, want := TargetPath("/src", repo, false), filepath.Join("/src", "api"); got != want {
		t.Errorf("TargetPath(orgSubdir=false) = %q, want %q", got, want)
	}
	if got, want := TargetPath("/src", repo, true), filepath.Join("/src", "acme", "api"); got != want {
		t.Errorf("TargetPath(orgSubdir=true) = %q, want %q", got, want)
	}
}

func TestCloneOne_FreshPathClonesSuccessfully(t *testing.T) {
	bare := newBareRepo(t)
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: bare}

	outcome, dest, err := CloneOne(context.Background(), target, repo, false)
	if err != nil {
		t.Fatalf("CloneOne() error = %v", err)
	}
	if outcome != OutcomeCloned {
		t.Errorf("Outcome = %v, want Cloned", outcome)
	}
	wantDest := filepath.Join(target, "api")
	if dest != wantDest {
		t.Errorf("dest = %q, want %q", dest, wantDest)
	}
	if _, err := os.Stat(filepath.Join(dest, ".git")); err != nil {
		t.Errorf("expected a .git directory at %s: %v", dest, err)
	}
}

func TestCloneOne_OrgSubdirTrueNestsUnderOrg(t *testing.T) {
	bare := newBareRepo(t)
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: bare}

	outcome, dest, err := CloneOne(context.Background(), target, repo, true)
	if err != nil {
		t.Fatalf("CloneOne() error = %v", err)
	}
	if outcome != OutcomeCloned {
		t.Errorf("Outcome = %v, want Cloned", outcome)
	}
	wantDest := filepath.Join(target, "acme", "api")
	if dest != wantDest {
		t.Errorf("dest = %q, want %q", dest, wantDest)
	}
	if _, err := os.Stat(filepath.Join(dest, ".git")); err != nil {
		t.Errorf("expected a .git directory at %s: %v", dest, err)
	}
}

func TestCloneOne_GitNotInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: "https://example.invalid/acme/api.git"}

	_, _, err := CloneOne(context.Background(), target, repo, false)
	if err == nil {
		t.Fatal("CloneOne() error = nil, want an error")
	}
	if !errors.Is(err, ErrGitNotInstalled) {
		t.Errorf("error = %v, want it to wrap ErrGitNotInstalled", err)
	}
}
