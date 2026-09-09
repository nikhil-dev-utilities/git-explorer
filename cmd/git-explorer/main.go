// Command git-explorer is the composition root: it wires the real gh-cli Forge
// adapter, resolved Config, and clone.Classify/clone.Run into internal/tui.Model and
// runs the Bubble Tea program. See DESIGN.md and CONTEXT.md for the vocabulary and
// design this wiring implements.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge/github"
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
	// can show if git or gh don't exist, and this is the one startup path allowed to
	// print directly rather than go through the log file — no render surface exists
	// yet to corrupt.
	if err := checkPrerequisites(); err != nil {
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

	// Only when the user's own config didn't declare hosts: — an explicit list
	// always wins over discovery, even if it happens to match what discovery would
	// have found anyway. hostsUserConfigured is also handed to buildModel: a Fatal
	// "not authenticated" failure against a discovered (or last-resort implicit)
	// Host is treated more gently than one against a Host the user explicitly
	// configured themselves — see tui.Model's hostsUserConfigured field.
	hostsUserConfigured := fileDeclaresHosts(fileBytes)
	if !hostsUserConfigured {
		if discovered := github.DiscoverAuthenticatedHosts(env.Getenv); len(discovered) > 0 {
			cfg.Hosts = discoveredConfigHosts(discovered, cfg.Clone.DefaultTarget)
		}
	}

	slog.SetDefault(config.NewLogger(cfg.Log))

	f := github.New()
	model := buildModel(f, cfg, hostsUserConfigured, previewClones, clone.Run)

	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}
