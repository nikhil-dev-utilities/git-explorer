package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// starterConfigContent is what gets written when no config.yaml exists yet — the
// full schema, commented out line by line, so config.Load resolves it identically
// to the file-absent case (YAML comments produce no parsed nodes) until the user
// actually uncomments or edits something. Kept in sync with DESIGN.md's own Config
// section example by hand — no test enforces this (unlike the README's keybindings
// table, there's no single source to drift-check a comment block against), so
// treat an edit to one as a prompt to check the other.
const starterConfigContent = `# git-explorer config.
#
# This file was created automatically because none existed yet. It is otherwise
# inert — every line below is commented out, so nothing here changes
# git-explorer's behavior until you uncomment or edit something. Full schema:

# clone:
#   default_target: ~/src       # pre-fills the clone dialog; always editable there too
#   parallelism: 8               # max concurrent "git clone" processes in a Clone Run

# log:
#   path: ~/.config/git-explorer/git-explorer.log   # default shown; --log-file overrides
#   level: info                  # debug | info | warn | error | off
#   max_size_mb: 5               # log file is rotated once it reaches this size

# hosts:                         # omit entirely to auto-detect from gh's own auth state
#   - name: github.com
#     frontdoor: gh-cli
#     protocol: ssh               # ssh | https
#   - name: ghe.corp.internal    # a self-managed GitHub Enterprise instance
#     frontdoor: gh-cli
#     protocol: https
#     default_target: ~/work     # overrides clone.default_target for this Host only

# See DESIGN.md's Config section (or the README) for the full write-up.
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
