package clone

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestCloneOne_ShallowFetchesOnlyTheTipCommit(t *testing.T) {
	work := t.TempDir()
	mustRunGit(t, "init", "-q", work)
	for _, msg := range []string{"one", "two"} {
		mustRunGit(t, "-C", work, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", msg)
	}
	bare := filepath.Join(t.TempDir(), "repo.git")
	mustRunGit(t, "clone", "-q", "--bare", work, bare)

	// --depth is ignored for plain local paths; file:// makes git honour it.
	repo := Repo{Org: "acme", Name: "api", CloneURL: "file://" + bare, Shallow: true}
	_, dest, err := CloneOne(context.Background(), t.TempDir(), repo, false)
	if err != nil {
		t.Fatalf("CloneOne() error = %v", err)
	}
	res, err := runGit(context.Background(), "-C", dest, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(res.Stdout)); got != "1" {
		t.Errorf("shallow clone has %s commits, want 1", got)
	}
}
