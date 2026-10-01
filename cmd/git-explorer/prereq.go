package main

import (
	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// lookPathFunc is exec.LookPath, injectable so tests need no fake binaries.
type lookPathFunc func(file string) (string, error)

// checkGit verifies git is on PATH before anything else runs — DESIGN.md commits to
// this as an actionable failure, not a late one three steps into using the app. It
// returns the same *forge.Error{Kind: ErrKindFatal} shape internal/forge/github already
// uses for a missing gh, so this startup failure and a later runtime one render
// identically.
func checkGit(lookPath lookPathFunc) error {
	if _, err := lookPath("git"); err != nil {
		return &forge.Error{
			Kind:    forge.ErrKindFatal,
			Message: "git-explorer requires git, but it was not found on PATH. Install git and try again.",
			Err:     err,
		}
	}
	return nil
}

// checkGh verifies gh is on PATH, but only when some Host is served by the GitHub
// Forge — a Bitbucket-only config never needs gh (ADR-0010). It runs once the Hosts
// are resolved; gh discovery only reads gh's hosts file, so it is safe before this.
func checkGh(hosts []config.HostConfig, lookPath lookPathFunc) error {
	for _, h := range hosts {
		if h.Forge != "github" {
			continue
		}
		if _, err := lookPath("gh"); err != nil {
			return &forge.Error{
				Kind:    forge.ErrKindFatal,
				Message: "git-explorer requires the GitHub CLI (gh) for GitHub Hosts, but it was not found on PATH. Install it from https://cli.github.com and try again.",
				Err:     err,
			}
		}
		return nil
	}
	return nil
}
