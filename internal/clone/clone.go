// Package clone implements the pre-flight Outcome check and the actual `git clone`
// execution for a Clone Run. It owns no dialog or progress-screen rendering — that
// belongs to internal/tui — and it never imports internal/forge: callers resolve each
// Repo's CloneURL once (via forge.Forge.CloneURL) and pass the result in. See
// DESIGN.md's "Clone" section and ADR-0003.
package clone

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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
// destination path that decision is about: Cloned (the path is absent), Skipped (the
// path holds a git repository whose origin remote identifies this same Repo — left
// entirely untouched), or Conflict (the path is occupied by anything else). A Clone
// Run must never write into a Conflict path, under any circumstance.
//
// The Skipped comparison is protocol-normalized: the existing origin and repo.CloneURL
// are each parsed into a (host, owner, name) identity and compared as that triple, not
// as raw strings, so an ssh clone and an https clone of the same Repo are recognized
// as the same Repo.
func Classify(ctx context.Context, target string, repo Repo, orgSubdir bool) (Outcome, string, error) {
	dest := TargetPath(target, repo, orgSubdir)

	if _, err := os.Stat(dest); err != nil {
		if os.IsNotExist(err) {
			return OutcomeCloned, dest, nil
		}
		return 0, "", fmt.Errorf("checking %s: %w", dest, err)
	}

	res, err := runGit(ctx, "-C", dest, "remote", "get-url", "origin")
	if err != nil {
		if errors.Is(err, ErrGitNotInstalled) {
			return 0, "", err
		}
		// Not a git repository, or a git repository with no origin remote, or any
		// other git failure: we cannot confirm this is the same Repo, so treat it
		// as occupied rather than guessing.
		return OutcomeConflict, dest, nil
	}

	existing, ok1 := parseCloneURL(strings.TrimSpace(string(res.Stdout)))
	wanted, ok2 := parseCloneURL(repo.CloneURL)
	if ok1 && ok2 && existing == wanted {
		return OutcomeSkipped, dest, nil
	}
	return OutcomeConflict, dest, nil
}

// CloneOne classifies repo against target and, when the Outcome is Cloned, performs
// the actual `git clone`. It returns the Outcome and destination path regardless of
// whether a clone was needed.
func CloneOne(ctx context.Context, target string, repo Repo, orgSubdir bool) (Outcome, string, error) {
	outcome, dest, err := Classify(ctx, target, repo, orgSubdir)
	if err != nil {
		return outcome, dest, err
	}
	if outcome != OutcomeCloned {
		return outcome, dest, nil
	}
	if err := performClone(ctx, repo.CloneURL, dest); err != nil {
		return outcome, dest, err
	}
	return outcome, dest, nil
}

// performClone is the actual `git clone` step, shared by CloneOne and Run so the two
// never duplicate — or drift apart on — how a clone is actually executed.
func performClone(ctx context.Context, cloneURL, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating parent directory for %s: %w", dest, err)
	}

	atomic.AddInt64(&activeClones, 1)
	defer atomic.AddInt64(&activeClones, -1)

	if _, err := runGit(ctx, "clone", "--origin", "origin", cloneURL, dest); err != nil {
		return err
	}
	return nil
}

// activeClones counts git clone subprocesses currently in flight. It exists so Run's
// parallelism bound can be verified by a test against real git — a counting wrapper
// around the exec calls, per this PRD's testing decisions — rather than by mocking
// git or asserting on wall-clock timing.
var activeClones int64
