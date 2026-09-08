package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// starterConfigContent is what gets written when no config.yaml exists yet — a
// header comment only, so config.Load resolves it identically to the file-absent
// case (YAML comments produce no parsed nodes) until the user actually edits it.
const starterConfigContent = `# git-explorer config.
#
# This file was created automatically because none existed yet. It is otherwise
# inert — nothing here changes git-explorer's behavior until you uncomment or add
# settings below. See DESIGN.md's Config section (or the README) for the full
# schema: clone.default_target, clone.parallelism, log.*, and hosts.
`

// bootstrapConfigDir ensures the directory configPath lives in exists, and writes a
// starter config.yaml there if none exists yet — this is git-explorer's own
// first-run convenience, reversing an earlier "never write config ourselves" design
// stance for the sake of a first-run user having somewhere obvious to look and edit.
// An existing file, of any content, is never touched or overwritten.
func bootstrapConfigDir(configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	_, err := os.Stat(configPath)
	if err == nil {
		return nil // already exists — never overwritten
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking for existing config file %s: %w", configPath, err)
	}

	if err := os.WriteFile(configPath, []byte(starterConfigContent), 0o644); err != nil {
		return fmt.Errorf("writing starter config file %s: %w", configPath, err)
	}
	return nil
}
