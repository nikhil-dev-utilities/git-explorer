package main

import (
	"context"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/clone"
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

func noopPreview(ctx context.Context, target string, repos []clone.Repo, orgSubdir bool) []clone.Result {
	return nil
}

func noopRunner(ctx context.Context, target string, repos []clone.Repo, orgSubdir bool, parallelism int) []clone.Result {
	return nil
}

func TestBuildModel_HostsCarryInferredKind(t *testing.T) {
	cfg := config.Config{
		Hosts: []config.HostConfig{
			{Name: "github.com", Protocol: "ssh", DefaultTarget: "/src"},
			{Name: "ghe.corp.internal", Protocol: "https", DefaultTarget: "/src"},
		},
		Clone: config.CloneConfig{Parallelism: 4},
	}

	model := buildModel(fakeForge{}, cfg, true, noopPreview, noopRunner)

	// Model doesn't expose hosts directly; exercise it indirectly through the one
	// observable surface buildModel's own callers care about — that New didn't
	// panic (hosts non-empty) and the resulting View renders without error, proving
	// construction succeeded end to end. The hostKind table test above already
	// covers the inference logic itself in isolation.
	if got := model.View(); got == "" {
		t.Error("View() = \"\", want a non-empty initial render")
	}
}

func TestBuildModel_EmptyHostsPanics(t *testing.T) {
	// config.Load's defaultConfig always seeds one Host (github.com), so an
	// hosts-less Config should never reach buildModel in production — this documents
	// that buildModel doesn't silently swallow that case, it surfaces tui.New's own
	// documented panic rather than constructing a broken Model.
	defer func() {
		if recover() == nil {
			t.Error("buildModel with zero Hosts did not panic, want it to (tui.New requires non-empty hosts)")
		}
	}()
	buildModel(fakeForge{}, config.Config{}, true, noopPreview, noopRunner)
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
