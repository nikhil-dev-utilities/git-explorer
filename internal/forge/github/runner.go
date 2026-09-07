package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// errGhNotFound is returned by runner.Run when the gh process could not even be
// started — e.g. gh is not on PATH. It is distinct from a non-zero exit code, which is
// reported via runResult.ExitCode instead, since a process that ran and failed is a
// different condition from a process that never ran at all.
var errGhNotFound = errors.New("gh: executable not found")

// runResult is the outcome of a gh invocation that actually ran, whether it succeeded
// or not.
type runResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// runner is the one seam this whole package is built around: every interaction with gh
// goes through it, whether that's an auth check or a data-fetching call. The real
// implementation shells out; tests inject a fake. Prefer extending this seam over
// introducing a second one (e.g. an HTTP mock), even for a future Frontdoor.
type runner interface {
	Run(ctx context.Context, args ...string) (runResult, error)
}

// execRunner is the real runner, backed by the gh binary on PATH.
type execRunner struct{}

func (execRunner) Run(ctx context.Context, args ...string) (runResult, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return runResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: 0}, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return runResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: exitErr.ExitCode()}, nil
	}

	// The process never ran at all (most commonly: gh isn't installed).
	return runResult{}, fmt.Errorf("%w: %v", errGhNotFound, err)
}
