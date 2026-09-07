package main

import (
	"os/exec"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// checkPrerequisites verifies git and gh are both on PATH before anything else runs —
// DESIGN.md commits to this as an actionable failure, not a late one three steps into
// using the app. It returns the same *forge.Error{Kind: ErrKindFatal} shape
// internal/forge/github already uses for a missing gh, so this startup failure and a
// later runtime one render identically.
func checkPrerequisites() error {
	if _, err := exec.LookPath("git"); err != nil {
		return &forge.Error{
			Kind:    forge.ErrKindFatal,
			Message: "git-explorer requires git, but it was not found on PATH. Install git and try again.",
			Err:     err,
		}
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return &forge.Error{
			Kind:    forge.ErrKindFatal,
			Message: "git-explorer requires the GitHub CLI (gh), but it was not found on PATH. Install it from https://cli.github.com and try again.",
			Err:     err,
		}
	}
	return nil
}
