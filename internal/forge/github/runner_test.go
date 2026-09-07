package github

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeFakeGh puts an executable named "gh" on an isolated PATH that writeFakeGh
// returns, so execRunner's real subprocess-handling code (exec.CommandContext, exit
// code translation, stdout/stderr capture) is exercised without depending on the real
// gh CLI being installed, authenticated, or reachable over the network — these tests
// stay hermetic per the PRD's "no real gh binary or network access" requirement.
func writeFakeGh(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake gh script is a POSIX shell script; Windows isn't a v1 release target")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "gh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatalf("writing fake gh script: %v", err)
	}
	return dir
}

func TestExecRunner_Success(t *testing.T) {
	dir := writeFakeGh(t, `echo "gh version 2.99.0"; exit 0`)
	t.Setenv("PATH", dir)

	res, err := execRunner{}.Run(context.Background(), "--version")
	if err != nil {
		t.Fatalf("Run returned an error for a process that should have started: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stderr = %q", res.ExitCode, res.Stderr)
	}
	if string(res.Stdout) != "gh version 2.99.0\n" {
		t.Fatalf("Stdout = %q, want the fake script's output", res.Stdout)
	}
}

func TestExecRunner_NonZeroExit(t *testing.T) {
	dir := writeFakeGh(t, `echo "unknown command" >&2; exit 1`)
	t.Setenv("PATH", dir)

	res, err := execRunner{}.Run(context.Background(), "this-subcommand-does-not-exist")
	if err != nil {
		t.Fatalf("Run returned an error for a process that ran and exited non-zero: %v", err)
	}
	if res.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", res.ExitCode)
	}
	if string(res.Stderr) != "unknown command\n" {
		t.Fatalf("Stderr = %q, want the fake script's stderr", res.Stderr)
	}
}

func TestExecRunner_NotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := execRunner{}.Run(context.Background(), "--version")
	if err == nil {
		t.Fatal("Run returned no error with gh removed from PATH")
	}
	if !errors.Is(err, errGhNotFound) {
		t.Fatalf("error = %v, want it to wrap errGhNotFound", err)
	}
}
