package config

import (
	"testing"
)

func TestLoad_ZeroConfigDefaults(t *testing.T) {
	cfg, err := Load(Flags{}, MapEnviron{"HOME": "/home/nikhil"}, nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Clone.DefaultTarget != "" {
		t.Errorf("Clone.DefaultTarget = %q, want empty", cfg.Clone.DefaultTarget)
	}
	if cfg.Clone.Parallelism != 8 {
		t.Errorf("Clone.Parallelism = %d, want 8", cfg.Clone.Parallelism)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want info", cfg.Log.Level)
	}
	if cfg.Log.MaxSizeMB != 5 {
		t.Errorf("Log.MaxSizeMB = %d, want 5", cfg.Log.MaxSizeMB)
	}
	wantLogPath := "/home/nikhil/.config/git-explorer/git-explorer.log"
	if cfg.Log.Path != wantLogPath {
		t.Errorf("Log.Path = %q, want %q", cfg.Log.Path, wantLogPath)
	}

	if len(cfg.Hosts) != 1 {
		t.Fatalf("got %d Hosts, want exactly 1 implicit Host: %+v", len(cfg.Hosts), cfg.Hosts)
	}
	host := cfg.Hosts[0]
	if host.Name != "github.com" || host.Frontdoor != "gh-cli" || host.Protocol != "ssh" {
		t.Errorf("implicit Host = %+v, want {github.com gh-cli ssh}", host)
	}
	if host.DefaultTarget != "" {
		t.Errorf("implicit Host.DefaultTarget = %q, want empty", host.DefaultTarget)
	}
}

func TestLoad_NilFileBytesIsNotAnError(t *testing.T) {
	// Absence of a config file is the documented normal case, not an error condition
	// — Load must not require a caller to special-case "file doesn't exist".
	if _, err := Load(Flags{}, MapEnviron{}, nil); err != nil {
		t.Fatalf("Load() with no file present returned an error: %v", err)
	}
}

func TestLoad_XDGConfigHomeWinsOverHOMEFallback(t *testing.T) {
	env := MapEnviron{"XDG_CONFIG_HOME": "/xdg-config", "HOME": "/home/nikhil"}
	cfg, err := Load(Flags{}, env, nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := "/xdg-config/git-explorer/git-explorer.log"
	if cfg.Log.Path != want {
		t.Errorf("Log.Path = %q, want %q (XDG_CONFIG_HOME set means HOME is never consulted)", cfg.Log.Path, want)
	}
}
