package clone

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func mustRunGitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	fullArgs := append([]string{"-C", dir}, args...)
	if _, err := runGit(context.Background(), fullArgs...); err != nil {
		t.Fatalf("git %v: %v", fullArgs, err)
	}
}

// initRepoWithOrigin creates a real (non-bare) git repository at dir with an origin
// remote pointed at originURL — enough for Classify's `git remote get-url origin`
// check, without the overhead of a full clone chain for every test case.
func initRepoWithOrigin(t *testing.T, dir, originURL string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustRunGitIn(t, dir, "init", "-q")
	mustRunGitIn(t, dir, "remote", "add", "origin", originURL)
}

func TestClassify_FreshPathIsCloned(t *testing.T) {
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: "https://github.com/acme/api.git"}

	outcome, _, err := Classify(context.Background(), target, repo, false)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if outcome != OutcomeCloned {
		t.Errorf("Outcome = %v, want Cloned", outcome)
	}
}

func TestClassify_SameRepoSameProtocolIsSkipped(t *testing.T) {
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: "https://github.com/acme/api.git"}
	initRepoWithOrigin(t, TargetPath(target, repo, false), "https://github.com/acme/api.git")

	outcome, _, err := Classify(context.Background(), target, repo, false)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if outcome != OutcomeSkipped {
		t.Errorf("Outcome = %v, want Skipped", outcome)
	}
}

func TestClassify_SameRepoOtherProtocolIsSkipped(t *testing.T) {
	// The trickiest case: existing clone is ssh, repo.CloneURL is now https (or vice
	// versa) — must still be recognized as the same Repo via protocol-normalized
	// identity comparison, not a raw string match.
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: "https://github.com/acme/api.git"}
	initRepoWithOrigin(t, TargetPath(target, repo, false), "git@github.com:acme/api.git")

	outcome, _, err := Classify(context.Background(), target, repo, false)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if outcome != OutcomeSkipped {
		t.Errorf("Outcome = %v, want Skipped (protocol-normalized identity match)", outcome)
	}
}

func TestClassify_DifferentRepoIsConflict(t *testing.T) {
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: "https://github.com/acme/api.git"}
	initRepoWithOrigin(t, TargetPath(target, repo, false), "https://github.com/globex/other.git")

	outcome, _, err := Classify(context.Background(), target, repo, false)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if outcome != OutcomeConflict {
		t.Errorf("Outcome = %v, want Conflict", outcome)
	}
}

func TestClassify_NonGitDirectoryIsConflict(t *testing.T) {
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: "https://github.com/acme/api.git"}
	dest := TargetPath(target, repo, false)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "scratch.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	outcome, _, err := Classify(context.Background(), target, repo, false)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if outcome != OutcomeConflict {
		t.Errorf("Outcome = %v, want Conflict", outcome)
	}
}

func TestClassify_PlainFileIsConflict(t *testing.T) {
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: "https://github.com/acme/api.git"}
	dest := TargetPath(target, repo, false)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	outcome, _, err := Classify(context.Background(), target, repo, false)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if outcome != OutcomeConflict {
		t.Errorf("Outcome = %v, want Conflict", outcome)
	}
}

func TestClassify_NeverModifiesAConflictPath(t *testing.T) {
	target := t.TempDir()
	repo := Repo{Org: "acme", Name: "api", CloneURL: "https://github.com/acme/api.git"}
	dest := TargetPath(target, repo, false)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dest, "dont-touch-me.txt")
	content := []byte("original content")
	if err := os.WriteFile(marker, content, 0o644); err != nil {
		t.Fatal(err)
	}

	outcome, _, err := Classify(context.Background(), target, repo, false)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if outcome != OutcomeConflict {
		t.Fatalf("Outcome = %v, want Conflict (to set up this test correctly)", outcome)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("reading marker file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("Conflict path was modified: got %q, want %q unchanged", got, content)
	}
}

func TestCloneOne_SkippedAndConflictNeverAttemptClone(t *testing.T) {
	t.Run("skipped", func(t *testing.T) {
		target := t.TempDir()
		// CloneURL points at an unreachable host (.invalid is reserved, never
		// resolves — RFC 2606). If CloneOne mistakenly attempted a clone despite
		// the Skipped outcome, it would fail fast here instead of this test
		// passing for the wrong reason.
		repo := Repo{Org: "acme", Name: "api", CloneURL: "https://example.invalid/acme/api.git"}
		initRepoWithOrigin(t, TargetPath(target, repo, false), "https://example.invalid/acme/api.git")

		outcome, _, err := CloneOne(context.Background(), target, repo, false)
		if err != nil {
			t.Fatalf("CloneOne() error = %v, want no clone attempt to have been made", err)
		}
		if outcome != OutcomeSkipped {
			t.Errorf("Outcome = %v, want Skipped", outcome)
		}
	})

	t.Run("conflict", func(t *testing.T) {
		target := t.TempDir()
		repo := Repo{Org: "acme", Name: "api", CloneURL: "https://example.invalid/acme/api.git"}
		dest := TargetPath(target, repo, false)
		if err := os.MkdirAll(dest, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dest, "scratch.txt"), []byte("hi"), 0o644); err != nil {
			t.Fatal(err)
		}

		outcome, _, err := CloneOne(context.Background(), target, repo, false)
		if err != nil {
			t.Fatalf("CloneOne() error = %v, want no clone attempt to have been made", err)
		}
		if outcome != OutcomeConflict {
			t.Errorf("Outcome = %v, want Conflict", outcome)
		}
	})
}
