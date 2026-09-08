package github

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// HostAuth is one Host gh's own config reports being authenticated to.
type HostAuth struct {
	Name string
	// GitProtocol is gh's recorded git_protocol for this Host ("ssh" or "https"),
	// or empty if gh's file didn't record one.
	GitProtocol string
}

// hostsFileEntry only ever defines git_protocol. gh's hosts.yml can carry other
// fields per host — user, users, and (on installs that don't use the system
// keychain) oauth_token, which is the actual credential. None of those are ever
// unmarshaled into anything: a field yaml.Unmarshal has nowhere to put is simply
// skipped, so oauth_token can never end up in a HostAuth, a log line, or anywhere
// else this package touches. See ADR-0004.
type hostsFileEntry struct {
	GitProtocol string `yaml:"git_protocol"`
}

// DiscoverAuthenticatedHosts reads gh's own hosts.yml to find which Host(s) the
// user is actually authenticated to, for composition-root use as the zero-config
// default Host list instead of a hardcoded github.com. Deliberately reads gh's
// config file directly rather than shelling out to `gh auth status` and parsing
// its human-oriented text output, which is not designed to be a stable parsing
// target across gh versions.
//
// Returns nil — not an error — when the file is missing, empty, or unreadable, or
// its YAML doesn't parse: discovery failing is never fatal, callers fall back to
// their own default. Results are sorted by Name for deterministic ordering.
func DiscoverAuthenticatedHosts(getenv func(string) string) []HostAuth {
	data, err := os.ReadFile(hostsFilePath(getenv))
	if err != nil {
		return nil
	}

	var raw map[string]hostsFileEntry
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}

	hosts := make([]HostAuth, 0, len(raw))
	for name, entry := range raw {
		hosts = append(hosts, HostAuth{Name: name, GitProtocol: entry.GitProtocol})
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })
	return hosts
}

// hostsFilePath mirrors gh's own config-directory resolution order (see
// `gh help environment`): $GH_CONFIG_DIR > $XDG_CONFIG_HOME/gh > $HOME/.config/gh.
func hostsFilePath(getenv func(string) string) string {
	dir := getenv("GH_CONFIG_DIR")
	if dir == "" {
		if xdg := getenv("XDG_CONFIG_HOME"); xdg != "" {
			dir = filepath.Join(xdg, "gh")
		} else {
			dir = filepath.Join(getenv("HOME"), ".config", "gh")
		}
	}
	return filepath.Join(dir, "hosts.yml")
}
