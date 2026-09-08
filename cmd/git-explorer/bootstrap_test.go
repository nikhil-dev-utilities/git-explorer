package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
)

func TestBootstrapConfigDir_CreatesDirAndStarterFile(t *testing.T) {
	base := t.TempDir()
	configPath := filepath.Join(base, "git-explorer", "config.yaml")

	if err := bootstrapConfigDir(configPath); err != nil {
		t.Fatalf("bootstrapConfigDir() error = %v", err)
	}

	if info, err := os.Stat(filepath.Dir(configPath)); err != nil || !info.IsDir() {
		t.Fatalf("directory %s was not created: %v", filepath.Dir(configPath), err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading starter config: %v", err)
	}
	if string(got) != starterConfigContent {
		t.Errorf("starter config content = %q, want %q", got, starterConfigContent)
	}
}

func TestBootstrapConfigDir_StarterFileResolvesLikeNoFileAtAll(t *testing.T) {
	base := t.TempDir()
	configPath := filepath.Join(base, "git-explorer", "config.yaml")

	if err := bootstrapConfigDir(configPath); err != nil {
		t.Fatalf("bootstrapConfigDir() error = %v", err)
	}

	fileBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading starter config: %v", err)
	}

	withFile, err := config.Load(config.Flags{}, config.MapEnviron{"HOME": "/home/nikhil"}, fileBytes)
	if err != nil {
		t.Fatalf("Load() with starter file error = %v", err)
	}
	withoutFile, err := config.Load(config.Flags{}, config.MapEnviron{"HOME": "/home/nikhil"}, nil)
	if err != nil {
		t.Fatalf("Load() with no file error = %v", err)
	}

	if !reflect.DeepEqual(withFile, withoutFile) {
		t.Errorf("Load() with starter file = %+v, want identical to no-file case %+v", withFile, withoutFile)
	}
}

func TestBootstrapConfigDir_NeverOverwritesAnExistingFile(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "git-explorer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("setup MkdirAll: %v", err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	const existing = "hosts:\n  - name: ghe.corp.internal\n"
	if err := os.WriteFile(configPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("setup WriteFile: %v", err)
	}

	if err := bootstrapConfigDir(configPath); err != nil {
		t.Fatalf("bootstrapConfigDir() error = %v", err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if string(got) != existing {
		t.Errorf("existing config content = %q, want untouched %q", got, existing)
	}
}

func TestBootstrapConfigDir_DirectoryAlreadyExists(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "git-explorer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("setup MkdirAll: %v", err)
	}
	configPath := filepath.Join(dir, "config.yaml")

	if err := bootstrapConfigDir(configPath); err != nil {
		t.Fatalf("bootstrapConfigDir() error = %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("starter config should still be written when only the dir pre-existed: %v", err)
	}
}
