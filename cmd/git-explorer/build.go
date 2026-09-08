package main

import (
	"path/filepath"

	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
	"github.com/nikhil-dev-utilities/git-explorer/internal/tui"
)

// buildModel constructs a tui.Model from an already-resolved Config, a real (or fake,
// in tests) forge.Forge, and the two composition-root adapters. It performs no
// filesystem, network, or environment access of its own, which is what makes it
// directly unit-testable. hostsUserConfigured is threaded straight through to
// tui.New — see that field's own doc comment on tui.Model.
func buildModel(f forge.Forge, cfg config.Config, hostsUserConfigured bool, preview tui.ClonePreviewFunc, runner tui.CloneRunnerFunc) tui.Model {
	hosts := make([]forge.Host, len(cfg.Hosts))
	for i, h := range cfg.Hosts {
		hosts[i] = forge.Host{
			Name:     h.Name,
			Kind:     hostKind(h.Name),
			Protocol: h.Protocol,
		}
	}

	// tui.New always starts with hosts[0] active (see its own doc comment), so that
	// Host's already-resolved DefaultTarget (config.Load folds the global
	// clone.default_target fallback into every Host — see resolveHostDefaultTargets)
	// is the one target pre-filling the clone dialog at launch.
	var target string
	if len(cfg.Hosts) > 0 {
		target = cfg.Hosts[0].DefaultTarget
	}

	return tui.New(f, hosts, hostsUserConfigured, preview, runner, target, cfg.Clone.Parallelism)
}

// hostKind infers a Host's Kind from its name. config.HostConfig has no Kind field of
// its own — DESIGN.md's config examples never included one, so the divergent
// Org-discovery behavior ADR-0002 requires is derived here, not configured by the user.
func hostKind(name string) forge.HostKind {
	if name == "github.com" {
		return forge.HostPublic
	}
	return forge.HostPrivate
}

// resolveConfigPath is --config > $XDG_CONFIG_HOME/git-explorer/config.yaml >
// ~/.config/git-explorer/config.yaml, per DESIGN.md's Config section. Pure function of
// its inputs, mirroring config.resolveLogPath's own precedence-chain style.
func resolveConfigPath(flags config.Flags, env config.Environ) string {
	if flags.ConfigPath != "" {
		return flags.ConfigPath
	}
	configHome := env.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(env.Getenv("HOME"), ".config")
	}
	return filepath.Join(configHome, "git-explorer", "config.yaml")
}
