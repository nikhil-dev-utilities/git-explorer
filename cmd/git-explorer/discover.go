package main

import (
	"gopkg.in/yaml.v3"

	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge/github"
)

// fileDeclaresHosts reports whether the raw config file bytes set a top-level
// hosts: key at all — not what config.Load resolved cfg.Hosts to, which is always
// non-empty (either the file's own value or the built-in single-github.com
// default) and so can't by itself distinguish "the user configured this" from
// "zero-config". An explicit hosts: list, even one that happens to match the
// built-in default, always wins over discovery; this is the one place that
// decision is made.
func fileDeclaresHosts(fileBytes []byte) bool {
	if len(fileBytes) == 0 {
		return false
	}
	var probe struct {
		Hosts *[]any `yaml:"hosts"`
	}
	if err := yaml.Unmarshal(fileBytes, &probe); err != nil {
		return false
	}
	return probe.Hosts != nil
}

// resolveHosts combines gh-discovered Hosts with the file's hosts: list (ADR-0013).
// Discovered Hosts come first, so adding a Host never changes the one that opens at
// startup. A file entry with a discovered Host's name replaces it in place; the other
// file entries follow in file order. With no hosts: key and nothing discovered, the
// file Hosts are config.Load's implicit github.com default. configured holds the names
// that came from the file.
func resolveHosts(discovered, fileHosts []config.HostConfig, declared bool) (hosts []config.HostConfig, configured map[string]bool) {
	configured = map[string]bool{}
	if !declared {
		if len(discovered) == 0 {
			return fileHosts, configured
		}
		return discovered, configured
	}
	pending := map[string]config.HostConfig{}
	for _, h := range fileHosts {
		pending[h.Name] = h
		configured[h.Name] = true
	}
	for _, h := range discovered {
		if f, ok := pending[h.Name]; ok {
			h = f
			delete(pending, h.Name)
		}
		hosts = append(hosts, h)
	}
	for _, h := range fileHosts {
		if _, ok := pending[h.Name]; ok {
			hosts = append(hosts, h)
			delete(pending, h.Name)
		}
	}
	return hosts, configured
}

// discoveredConfigHosts converts gh's own authenticated-host list into the
// []config.HostConfig shape the rest of the composition root already works with,
// applying the same DefaultTarget-fallback rule config.Load's own
// resolveHostDefaultTargets applies to every other Host (unexported there, so
// reimplemented here rather than exported just for this one caller) and defaulting
// Protocol to "ssh" when gh's file didn't record a git_protocol.
func discoveredConfigHosts(auths []github.HostAuth, fallbackTarget string) []config.HostConfig {
	hosts := make([]config.HostConfig, len(auths))
	for i, a := range auths {
		protocol := a.GitProtocol
		if protocol == "" {
			protocol = "ssh"
		}
		hosts[i] = config.HostConfig{
			Name:          a.Name,
			Forge:         "github",
			Frontdoor:     "gh-cli",
			Protocol:      protocol,
			DefaultTarget: fallbackTarget,
		}
	}
	return hosts
}
