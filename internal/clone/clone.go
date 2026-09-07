// Package clone implements the pre-flight Outcome check and the actual `git clone`
// execution for a Clone Run. It owns no dialog or progress-screen rendering — that
// belongs to internal/tui — and it never imports internal/forge: callers resolve each
// Repo's CloneURL once (via forge.Forge.CloneURL) and pass the result in. See
// DESIGN.md's "Clone" section and ADR-0003.
package clone

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Repo is the minimal input this package needs — deliberately decoupled from
// forge.Repo, which carries fields (Affiliation, Visibility, PushedAt, ...) this
// package has no use for.
type Repo struct {
	Org      string
	Name     string
	CloneURL string
}

// Outcome is what happens to a Repo when it's classified against a Target.
type Outcome int

const (
	// OutcomeCloned means the Target path was absent; it was (or will be) cloned.
	OutcomeCloned Outcome = iota
	// OutcomeSkipped means the Target path already holds this same Repo; it is left
	// entirely untouched.
	OutcomeSkipped
	// OutcomeConflict means the Target path is occupied by anything else. A Clone
	// Run never writes into a Conflict path, under any circumstance.
	OutcomeConflict
)

func (o Outcome) String() string {
	switch o {
	case OutcomeSkipped:
		return "skipped"
	case OutcomeConflict:
		return "conflict"
	default:
		return "cloned"
	}
}

// TargetPath returns the destination directory for repo under target. With orgSubdir
// off (the default, and never remembered between runs — ADR-0007), it's
// target/<name>; with it on, target/<org>/<name>. This is pure path computation: it
// reads and writes nothing.
func TargetPath(target string, repo Repo, orgSubdir bool) string {
	if orgSubdir {
		return filepath.Join(target, repo.Org, repo.Name)
	}
	return filepath.Join(target, repo.Name)
}

// Classify determines what will happen to repo if cloned to target, and the
// destination path that decision is about.
//
// This slice only implements the fresh-path case: an absent destination always
// classifies Cloned. Recognizing an existing clone of the same Repo as Skipped, or
// anything else at the path as Conflict, is a later slice of this PRD (#18) — calling
// Classify against an already-occupied path returns an error rather than a wrong
// Outcome in the meantime.
func Classify(target string, repo Repo, orgSubdir bool) (Outcome, string, error) {
	dest := TargetPath(target, repo, orgSubdir)

	if _, err := os.Stat(dest); err != nil {
		if os.IsNotExist(err) {
			return OutcomeCloned, dest, nil
		}
		return 0, "", fmt.Errorf("checking %s: %w", dest, err)
	}

	return 0, "", fmt.Errorf("clone: classifying an existing path at %s is not implemented yet", dest)
}

// CloneOne classifies repo against target and, when the Outcome is Cloned, performs
// the actual `git clone`. It returns the Outcome and destination path regardless of
// whether a clone was needed.
func CloneOne(ctx context.Context, target string, repo Repo, orgSubdir bool) (Outcome, string, error) {
	outcome, dest, err := Classify(target, repo, orgSubdir)
	if err != nil {
		return outcome, dest, err
	}
	if outcome != OutcomeCloned {
		return outcome, dest, nil
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return outcome, dest, fmt.Errorf("creating parent directory for %s: %w", dest, err)
	}
	if _, err := runGit(ctx, "clone", "--origin", "origin", repo.CloneURL, dest); err != nil {
		return outcome, dest, err
	}
	return outcome, dest, nil
}
