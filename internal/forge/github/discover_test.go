package github

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAuthenticatedHosts_ParsesNameAndGitProtocol(t *testing.T) {
	dir := t.TempDir()
	writeHostsFile(t, dir, `github.com:
    git_protocol: ssh
    user: fernandesnikhil
ghe.corp.internal:
    git_protocol: https
    user: fernandesnikhil
`)

	getenv := envMap(map[string]string{"GH_CONFIG_DIR": dir})
	got := DiscoverAuthenticatedHosts(getenv)

	want := []HostAuth{
		{Name: "ghe.corp.internal", GitProtocol: "https"},
		{Name: "github.com", GitProtocol: "ssh"},
	}
	assertHostAuthsEqual(t, got, want)
}

// A real observed hosts.yml on a keychain-backed install has no oauth_token field
// at all — this fixture matches that shape exactly, proving discovery works
// without ever needing one.
func TestDiscoverAuthenticatedHosts_NoOauthTokenFieldPresent(t *testing.T) {
	dir := t.TempDir()
	writeHostsFile(t, dir, `github.com:
    git_protocol: ssh
    users:
        fernandesnikhil:
    user: fernandesnikhil
`)

	getenv := envMap(map[string]string{"GH_CONFIG_DIR": dir})
	got := DiscoverAuthenticatedHosts(getenv)

	assertHostAuthsEqual(t, got, []HostAuth{{Name: "github.com", GitProtocol: "ssh"}})
}

func TestDiscoverAuthenticatedHosts_MissingFileReturnsNilNotError(t *testing.T) {
	dir := t.TempDir() // no hosts.yml written
	getenv := envMap(map[string]string{"GH_CONFIG_DIR": dir})

	got := DiscoverAuthenticatedHosts(getenv)
	if got != nil {
		t.Errorf("DiscoverAuthenticatedHosts() = %+v, want nil for a missing file", got)
	}
}

func TestDiscoverAuthenticatedHosts_UnparsableFileReturnsNil(t *testing.T) {
	dir := t.TempDir()
	writeHostsFile(t, dir, "not: [valid: yaml: at: all")

	getenv := envMap(map[string]string{"GH_CONFIG_DIR": dir})
	got := DiscoverAuthenticatedHosts(getenv)
	if got != nil {
		t.Errorf("DiscoverAuthenticatedHosts() = %+v, want nil for unparsable YAML", got)
	}
}

func TestHostsFilePath_PrecedenceOrder(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "GH_CONFIG_DIR wins",
			env:  map[string]string{"GH_CONFIG_DIR": "/ghconf", "XDG_CONFIG_HOME": "/xdg", "HOME": "/home/u"},
			want: "/ghconf/hosts.yml",
		},
		{
			name: "XDG_CONFIG_HOME when GH_CONFIG_DIR unset",
			env:  map[string]string{"XDG_CONFIG_HOME": "/xdg", "HOME": "/home/u"},
			want: "/xdg/gh/hosts.yml",
		},
		{
			name: "falls back to HOME/.config/gh",
			env:  map[string]string{"HOME": "/home/u"},
			want: "/home/u/.config/gh/hosts.yml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostsFilePath(envMap(tt.env)); got != tt.want {
				t.Errorf("hostsFilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func writeHostsFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "hosts.yml"), []byte(content), 0o600); err != nil {
		t.Fatalf("writing hosts.yml fixture: %v", err)
	}
}

func envMap(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

func assertHostAuthsEqual(t *testing.T, got, want []HostAuth) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d hosts, want %d: got=%+v want=%+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("host[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
