package github

import (
	"context"
	"errors"
	"testing"
)

// These tests exercise the real gh binary via execRunner, but only through
// subcommands that touch neither the network nor stored credentials (--version, and an
// invalid subcommand), so they stay hermetic per DESIGN.md's testing decisions.

func TestExecRunner_Success(t *testing.T) {
	res, err := execRunner{}.Run(context.Background(), "--version")
	if err != nil {
		t.Fatalf("Run returned an error for a process that should have started: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stderr = %q", res.ExitCode, res.Stderr)
	}
	if len(res.Stdout) == 0 {
		t.Fatal("Stdout is empty, want gh version output")
	}
}

func TestExecRunner_NonZeroExit(t *testing.T) {
	res, err := execRunner{}.Run(context.Background(), "this-subcommand-does-not-exist")
	if err != nil {
		t.Fatalf("Run returned an error for a process that ran and exited non-zero: %v", err)
	}
	if res.ExitCode == 0 {
		t.Fatal("ExitCode = 0, want non-zero for an invalid subcommand")
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
