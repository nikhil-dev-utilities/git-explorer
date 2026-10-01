package main

import (
	"context"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestHostKind(t *testing.T) {
	tests := []struct {
		name string
		host string
		want forge.HostKind
	}{
		{"public github.com", "github.com", forge.HostPublic},
		{"self-managed GHE", "ghe.corp.internal", forge.HostPrivate},
		{"arbitrary hostname", "git.example.org", forge.HostPrivate},
		{"public bitbucket.org", "bitbucket.org", forge.HostPublic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostKind(tt.host); got != tt.want {
				t.Errorf("hostKind(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

type fakeForge struct{}

func (fakeForge) ListOrgs(ctx context.Context, host forge.Host) (<-chan forge.OrgPage, error) {
	return nil, nil
}
func (fakeForge) ListRepos(ctx context.Context, org forge.Org) ([]forge.Repo, error) { return nil, nil }
func (fakeForge) CloneURL(repo forge.Repo) string                                    { return "" }

func TestBuildDeps_HostsKindsTargetAndClonerWiring(t *testing.T) {
	cfg := config.Config{
		Hosts: []config.HostConfig{
			{Name: "github.com", Protocol: "ssh", DefaultTarget: "/src"},
			{Name: "ghe.corp.internal", Forge: "github", Protocol: "https", DefaultTarget: "/work"},
		},
		Clone: config.CloneConfig{Parallelism: 4},
	}

	d := buildDeps(fakeForge{}, cfg, true)

	if len(d.Hosts) != 2 || d.Hosts[0].Kind != forge.HostPublic || d.Hosts[1].Kind != forge.HostPrivate ||
		d.Hosts[1].Protocol != "https" || d.Hosts[1].Forge != "github" {
		t.Errorf("Hosts = %+v", d.Hosts)
	}
	if d.CloneTarget != "/src" {
		t.Errorf("CloneTarget = %q, want the first Host's DefaultTarget", d.CloneTarget)
	}
	if d.Parallelism != 4 || !d.HostsUserConfigured {
		t.Errorf("Parallelism=%d HostsUserConfigured=%v", d.Parallelism, d.HostsUserConfigured)
	}
	if d.Preview == nil || d.Run == nil {
		t.Error("clone preview and runner must be wired")
	}
}

func TestResolveConfigPath(t *testing.T) {
	tests := []struct {
		name  string
		flags config.Flags
		env   config.MapEnviron
		want  string
	}{
		{
			name:  "flag wins",
			flags: config.Flags{ConfigPath: "/custom/config.yaml"},
			env:   config.MapEnviron{"XDG_CONFIG_HOME": "/xdg", "HOME": "/home/u"},
			want:  "/custom/config.yaml",
		},
		{
			name:  "XDG_CONFIG_HOME when set",
			flags: config.Flags{},
			env:   config.MapEnviron{"XDG_CONFIG_HOME": "/xdg", "HOME": "/home/u"},
			want:  "/xdg/git-explorer/config.yaml",
		},
		{
			name:  "falls back to HOME/.config",
			flags: config.Flags{},
			env:   config.MapEnviron{"HOME": "/home/u"},
			want:  "/home/u/.config/git-explorer/config.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveConfigPath(tt.flags, tt.env); got != tt.want {
				t.Errorf("resolveConfigPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
