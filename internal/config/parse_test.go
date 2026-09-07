package config

import (
	"strings"
	"testing"
)

func TestLoad_ParsesCloneAndHostFields(t *testing.T) {
	yaml := `
clone:
  default_target: ~/src
  parallelism: 4

hosts:
  - name: github.com
    frontdoor: gh-cli
    protocol: ssh
`
	cfg, err := Load(Flags{}, MapEnviron{"HOME": "/home/nikhil"}, []byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Clone.DefaultTarget != "~/src" {
		t.Errorf("Clone.DefaultTarget = %q, want ~/src", cfg.Clone.DefaultTarget)
	}
	if cfg.Clone.Parallelism != 4 {
		t.Errorf("Clone.Parallelism = %d, want 4", cfg.Clone.Parallelism)
	}
	if len(cfg.Hosts) != 1 {
		t.Fatalf("got %d Hosts, want 1: %+v", len(cfg.Hosts), cfg.Hosts)
	}
	got := cfg.Hosts[0]
	if got.Name != "github.com" || got.Frontdoor != "gh-cli" || got.Protocol != "ssh" {
		t.Errorf("Hosts[0] = %+v, want {github.com gh-cli ssh ...}", got)
	}
}

func TestLoad_HostDefaultTargetOverridesGlobal(t *testing.T) {
	yaml := `
clone:
  default_target: ~/src

hosts:
  - name: github.com
    frontdoor: gh-cli
    protocol: ssh
  - name: ghe.corp.internal
    frontdoor: gh-cli
    protocol: https
    default_target: ~/work
`
	cfg, err := Load(Flags{}, MapEnviron{"HOME": "/home/nikhil"}, []byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Hosts) != 2 {
		t.Fatalf("got %d Hosts, want 2 (multiple Hosts must not be collapsed): %+v", len(cfg.Hosts), cfg.Hosts)
	}

	byName := map[string]HostConfig{}
	for _, h := range cfg.Hosts {
		byName[h.Name] = h
	}

	// github.com set no override: it must fall back to the global default_target.
	if got := byName["github.com"].DefaultTarget; got != "~/src" {
		t.Errorf("github.com DefaultTarget = %q, want the global ~/src", got)
	}
	// ghe.corp.internal set its own: it must win over the global one, not be
	// overwritten by it.
	if got := byName["ghe.corp.internal"].DefaultTarget; got != "~/work" {
		t.Errorf("ghe.corp.internal DefaultTarget = %q, want its own override ~/work", got)
	}
}

func TestLoad_MalformedYAMLReturnsASpecificError(t *testing.T) {
	malformed := []byte("clone:\n  parallelism: [this is not a number\n")

	_, err := Load(Flags{ConfigPath: "/home/nikhil/.config/git-explorer/config.yaml"}, MapEnviron{}, malformed)
	if err == nil {
		t.Fatal("Load() error = nil, want a parse error")
	}
	if !strings.Contains(err.Error(), "/home/nikhil/.config/git-explorer/config.yaml") {
		t.Errorf("error = %q, want it to name the config file path", err.Error())
	}
}

func TestLoad_MalformedYAMLWithoutConfigPathStillErrorsClearly(t *testing.T) {
	malformed := []byte("clone:\n  parallelism: [this is not a number\n")

	_, err := Load(Flags{}, MapEnviron{}, malformed)
	if err == nil {
		t.Fatal("Load() error = nil, want a parse error")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error = %q, want it to say it's a config parse failure", err.Error())
	}
}
