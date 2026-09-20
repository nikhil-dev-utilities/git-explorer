package main

import (
	"path/filepath"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
	"github.com/nikhil-dev-utilities/git-explorer/internal/ui"
)

// buildDeps turns an already-resolved Config and a real (or fake, in tests) forge.Forge
// into the ui's dependencies, adding the composition-root pre-flight classifier and
// streaming Clone Run. It performs no filesystem, network, or environment access of its
// own, which is what makes it directly unit-testable. hostsUserConfigured says whether
// the Hosts came from an explicit hosts: list — see ui.Deps.
func buildDeps(f forge.Forge, cfg config.Config, hostsUserConfigured bool) ui.Deps {
	hosts := make([]forge.Host, len(cfg.Hosts))
	for i, h := range cfg.Hosts {
		hosts[i] = forge.Host{
			Name:     h.Name,
			Kind:     hostKind(h.Name),
			Protocol: h.Protocol,
		}
	}

	// The UI starts on hosts[0], so that Host's already-resolved DefaultTarget
	// (config.Load folds the global clone.default_target fallback into every Host — see
	// resolveHostDefaultTargets) is the Target the clone screen opens on.
	var target string
	if len(cfg.Hosts) > 0 {
		target = cfg.Hosts[0].DefaultTarget
	}

	return ui.Deps{
		Forge:               f,
		Hosts:               hosts,
		HostsUserConfigured: hostsUserConfigured,
		CloneTarget:         target,
		Parallelism:         cfg.Clone.Parallelism,
		Preview:             previewClones,
		Run:                 clone.RunProgress,
	}
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
