package config

import "testing"

// These tests exercise the log path precedence chain — --log-file flag > the
// GIT_EXPLORER_LOG env var > log.path in the config file > the XDG default — which
// resolveLogPath already implemented as the minimal correct way to compute the default
// (see #9). Each test proves one level of the chain winning over everything below it,
// with every lower-precedence source also populated so a bug that picks the wrong tier
// would actually be caught.

func TestLogPathPrecedence_FlagWinsOverEverything(t *testing.T) {
	cfg, err := Load(
		Flags{LogFile: "/from-flag.log"},
		MapEnviron{"GIT_EXPLORER_LOG": "/from-env.log", "HOME": "/home/nikhil"},
		[]byte("log:\n  path: /from-file.log\n"),
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Log.Path != "/from-flag.log" {
		t.Errorf("Log.Path = %q, want the flag value to win", cfg.Log.Path)
	}
}

func TestLogPathPrecedence_EnvWinsOverFileAndDefault(t *testing.T) {
	cfg, err := Load(
		Flags{},
		MapEnviron{"GIT_EXPLORER_LOG": "/from-env.log", "HOME": "/home/nikhil"},
		[]byte("log:\n  path: /from-file.log\n"),
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Log.Path != "/from-env.log" {
		t.Errorf("Log.Path = %q, want the env var to win (no flag set)", cfg.Log.Path)
	}
}

func TestLogPathPrecedence_FileWinsOverDefault(t *testing.T) {
	cfg, err := Load(
		Flags{},
		MapEnviron{"HOME": "/home/nikhil"},
		[]byte("log:\n  path: /from-file.log\n"),
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Log.Path != "/from-file.log" {
		t.Errorf("Log.Path = %q, want the file value to win (no flag, no env)", cfg.Log.Path)
	}
}

func TestLogPathPrecedence_DefaultAppliesWhenNothingElseIsSet(t *testing.T) {
	cfg, err := Load(Flags{}, MapEnviron{"HOME": "/home/nikhil"}, nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := "/home/nikhil/.local/state/git-explorer/git-explorer.log"
	if cfg.Log.Path != want {
		t.Errorf("Log.Path = %q, want the XDG default %q", cfg.Log.Path, want)
	}
}

func TestLoad_LogLevelAndMaxSizeFromFile(t *testing.T) {
	cfg, err := Load(Flags{}, MapEnviron{"HOME": "/home/nikhil"}, []byte("log:\n  level: debug\n  max_size_mb: 20\n"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want debug", cfg.Log.Level)
	}
	if cfg.Log.MaxSizeMB != 20 {
		t.Errorf("Log.MaxSizeMB = %d, want 20", cfg.Log.MaxSizeMB)
	}
}

func TestLoad_LogLevelAndMaxSizeDefaultWhenFileOmitsThem(t *testing.T) {
	// A file that sets something unrelated (here, just a Host) must not disturb the
	// documented log.level/log.max_size_mb defaults.
	cfg, err := Load(Flags{}, MapEnviron{"HOME": "/home/nikhil"}, []byte("hosts:\n  - name: github.com\n"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want the default info", cfg.Log.Level)
	}
	if cfg.Log.MaxSizeMB != 5 {
		t.Errorf("Log.MaxSizeMB = %d, want the default 5", cfg.Log.MaxSizeMB)
	}
}
