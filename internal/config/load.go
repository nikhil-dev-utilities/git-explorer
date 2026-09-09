package config

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Flags carries the command-line values that can override configuration, gathered by
// a thin wrapper (not part of this package) from the actual flag parser. An empty
// string means the flag was not set.
type Flags struct {
	// ConfigPath is --config: which file to read. Not used by Load itself (the
	// wrapper decides what to read using it and passes the result as fileBytes) —
	// kept here only so a future error path can name the file a parse failure came
	// from.
	ConfigPath string
	// LogFile is --log-file: the highest-precedence source for the log path.
	LogFile string
}

// Load resolves a Config from already-retrieved inputs — it never touches the
// filesystem or environment itself, which is what makes it trivially unit-testable.
// A thin wrapper (not part of this package) is responsible for the actual XDG lookup,
// environment reads, and file read.
//
// With empty fileBytes (no config file present, or none configured), Load returns the
// documented implicit defaults: one Host (github.com, gh-cli Frontdoor, ssh protocol),
// no default Target, info log level, parallelism 8, and the log path resolved against
// flags/env/XDG default (see resolveLogPath) — there being no config file at all is
// not an error.
func Load(flags Flags, env Environ, fileBytes []byte) (Config, error) {
	cfg := defaultConfig()

	if len(fileBytes) > 0 {
		if err := yaml.Unmarshal(fileBytes, &cfg); err != nil {
			return Config{}, parseError(flags.ConfigPath, err)
		}
	}

	cfg.Log.Path = resolveLogPath(flags, env, cfg.Log.Path)
	resolveHostDefaultTargets(&cfg)

	return cfg, nil
}

// parseError wraps a YAML parse failure, naming the file it came from when the caller
// told Load which file it read — yaml.v3's own error already includes a line number,
// which this preserves via %w.
func parseError(configPath string, cause error) error {
	if configPath == "" {
		return fmt.Errorf("parsing config: %w", cause)
	}
	return fmt.Errorf("parsing config file %s: %w", configPath, cause)
}

// resolveHostDefaultTargets fills in each Host's effective default clone Target: its
// own default_target if it set one, otherwise the global clone.default_target. Once
// this runs, a Host's DefaultTarget is always the final value — callers never need to
// know the fallback rule themselves.
func resolveHostDefaultTargets(cfg *Config) {
	for i := range cfg.Hosts {
		if cfg.Hosts[i].DefaultTarget == "" {
			cfg.Hosts[i].DefaultTarget = cfg.Clone.DefaultTarget
		}
	}
}

func defaultConfig() Config {
	return Config{
		Clone: CloneConfig{
			Parallelism: 8,
		},
		Log: LogConfig{
			Level:     "info",
			MaxSizeMB: 5,
		},
		Hosts: []HostConfig{
			{Name: "github.com", Frontdoor: "gh-cli", Protocol: "ssh"},
		},
	}
}

// resolveLogPath applies the log path precedence documented in DESIGN.md:
// --log-file flag > GIT_EXPLORER_LOG env var > log.path in the config file (already
// parsed into fileLogPath by the time this runs) > the default, alongside config.yaml.
func resolveLogPath(flags Flags, env Environ, fileLogPath string) string {
	if flags.LogFile != "" {
		return flags.LogFile
	}
	if v := env.Getenv("GIT_EXPLORER_LOG"); v != "" {
		return v
	}
	if fileLogPath != "" {
		return fileLogPath
	}
	return defaultLogPath(env)
}

// defaultLogPath is $HOME/.logs/git-explorer/git-explorer.log — a fixed path, not
// resolved against any XDG env var (unlike the config file's own location). Neither
// the binary's own directory nor the working directory is ever considered.
//
// This has moved twice: originally $XDG_STATE_HOME (strict XDG separation of state
// from config), then $XDG_CONFIG_HOME (alongside config.yaml, on the reasoning that
// one directory a user can find once beats XDG purity), now this literal path, by
// explicit request. The containing directory does not need to be created ahead of
// time — confirmed directly: lumberjack.Logger creates it (nested segments
// included) on first write, the same way it creates the log file itself.
func defaultLogPath(env Environ) string {
	return filepath.Join(env.Getenv("HOME"), ".logs", "git-explorer", "git-explorer.log")
}
