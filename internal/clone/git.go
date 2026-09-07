package clone

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrGitNotInstalled is returned (wrapped) when the git process could not be started
// at all — e.g. git is not on PATH. This is distinct from git running and exiting
// non-zero, which is a normal clone failure reported with its stderr instead.
var ErrGitNotInstalled = errors.New("git: executable not found")

type gitResult struct {
	Stdout []byte
	Stderr []byte
}

// runGit shells out to the real git binary (ADR-0003), inheriting the user's
// ssh-agent, credential helpers, url.insteadOf rewrites, and proxy/CA configuration
// rather than reimplementing any of it. Per this PRD's testing decisions, this
// package's tests exercise real git against local bare repositories — runGit has no
// fake/injectable seam.
func runGit(ctx context.Context, args ...string) (gitResult, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return gitResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return gitResult{}, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}

	// The process never ran at all (most commonly: git isn't installed).
	return gitResult{}, fmt.Errorf("%w: %v", ErrGitNotInstalled, err)
}
