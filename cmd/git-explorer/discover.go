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
			Frontdoor:     "gh-cli",
			Protocol:      protocol,
			DefaultTarget: fallbackTarget,
		}
	}
	return hosts
}
