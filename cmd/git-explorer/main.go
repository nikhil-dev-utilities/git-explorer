// Command git-explorer is the composition root: it wires the Forge adapters (behind a
// forgeRouter), resolved Config, and clone.Classify/clone.RunProgress into internal/ui and
// runs the Glyph app. See DESIGN.md and CONTEXT.md for the vocabulary and
// design this wiring implements.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge/bitbucket"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge/github"
	"github.com/nikhil-dev-utilities/git-explorer/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var flags config.Flags
	flag.StringVar(&flags.ConfigPath, "config", "", "path to config file (default: $XDG_CONFIG_HOME/git-explorer/config.yaml)")
	flag.StringVar(&flags.LogFile, "log-file", "", "path to log file (default: ~/.logs/git-explorer/git-explorer.log)")
	flag.Parse()

	// Checked before anything else is constructed: there is nothing useful the TUI
	// can show if git doesn't exist, and this is the one startup path allowed to
	// print directly rather than go through the log file — no render surface exists
	// yet to corrupt. gh is checked below, once the Hosts say whether it is needed.
	if err := checkGit(exec.LookPath); err != nil {
		return err
	}

	env := config.OSEnviron{}
	flags.ConfigPath = resolveConfigPath(flags, env)

	if err := bootstrapConfigDir(flags.ConfigPath); err != nil {
		return err
	}

	fileBytes, err := os.ReadFile(flags.ConfigPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading config file %s: %w", flags.ConfigPath, err)
	}

	cfg, err := config.Load(flags, env, fileBytes)
	if err != nil {
		return err
	}

	// gh-discovered Hosts and the file's hosts: list are combined (ADR-0013). The
	// configured set is handed to buildDeps: a Fatal "not authenticated" failure on a
	// Host the user did not list is treated more gently — see ui.Deps.ConfiguredHosts.
	var discovered []config.HostConfig
	if cfg.GhDiscovery {
		discovered = discoveredConfigHosts(github.DiscoverAuthenticatedHosts(env.Getenv), cfg.Clone.DefaultTarget)
	}
	var configured map[string]bool
	cfg.Hosts, configured = resolveHosts(discovered, cfg.Hosts, fileDeclaresHosts(fileBytes))

	if err := checkGh(cfg.Hosts, exec.LookPath); err != nil {
		return err
	}

	slog.SetDefault(config.NewLogger(cfg.Log))

	router := forgeRouter{"github": github.New(), "bitbucket": bitbucket.New()}
	return ui.Run(buildDeps(router, cfg, configured))
}
