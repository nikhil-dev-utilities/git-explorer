// Package config loads git-explorer's optional, non-secret configuration file and
// resolves logging settings. See ADR-0004 (never a credential) and DESIGN.md's "Config"
// and "Logging" sections.
package config

// Config is git-explorer's fully resolved configuration. Every field has a sensible
// value even when no config file exists — see Load.
type Config struct {
	Clone CloneConfig  `yaml:"clone"`
	Log   LogConfig    `yaml:"log"`
	Hosts []HostConfig `yaml:"hosts"`
}

// CloneConfig holds settings for the clone dialog and Clone Run.
type CloneConfig struct {
	// DefaultTarget pre-fills the clone dialog's target directory. It is never used
	// to clone without the user confirming the dialog — that guarantee is enforced
	// by whichever package owns the clone confirmation dialog, not by this package.
	DefaultTarget string `yaml:"default_target"`
	Parallelism   int    `yaml:"parallelism"`
}

// LogConfig holds the resolved logging settings. Path resolution (flag > env var >
// this file value > XDG default) is Load's job; by the time a Config is returned,
// Path already reflects the winning value.
type LogConfig struct {
	Path      string `yaml:"path"`
	Level     string `yaml:"level"`
	MaxSizeMB int    `yaml:"max_size_mb"`
}

// HostConfig declares one Host git-explorer can talk to.
type HostConfig struct {
	Name      string `yaml:"name"`
	Frontdoor string `yaml:"frontdoor"`
	Protocol  string `yaml:"protocol"`
	// DefaultTarget, when set, overrides CloneConfig.DefaultTarget for this Host
	// specifically.
	DefaultTarget string `yaml:"default_target"`
}
