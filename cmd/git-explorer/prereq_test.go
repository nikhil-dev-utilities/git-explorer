package main

import (
	"errors"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestCheckPrerequisites_BothPresent(t *testing.T) {
	// A dev/CI environment always has both on PATH — no fake-binary scaffolding
	// needed for the success path.
	if err := checkPrerequisites(); err != nil {
		t.Errorf("checkPrerequisites() = %v, want nil (git and gh are on PATH in this environment)", err)
	}
}

func TestCheckPrerequisites_NeitherOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := checkPrerequisites()
	if err == nil {
		t.Fatal("checkPrerequisites() = nil, want an error with git and gh both missing from PATH")
	}

	var fErr *forge.Error
	if !errors.As(err, &fErr) {
		t.Fatalf("error = %v (%T), want a *forge.Error", err, err)
	}
	if fErr.Kind != forge.ErrKindFatal {
		t.Errorf("Kind = %v, want ErrKindFatal", fErr.Kind)
	}
}
