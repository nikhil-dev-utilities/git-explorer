package main

import (
	"fmt"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge/github"
)

func TestFileDeclaresHosts(t *testing.T) {
	tests := []struct {
		name      string
		fileBytes []byte
		want      bool
	}{
		{"no file at all", nil, false},
		{"empty file", []byte(""), false},
		{"file sets unrelated keys only", []byte("clone:\n  parallelism: 4\n"), false},
		{"file declares hosts", []byte("hosts:\n  - name: github.com\n"), true},
		{
			"file declares hosts matching the built-in default exactly",
			[]byte("hosts:\n  - name: github.com\n    frontdoor: gh-cli\n    protocol: ssh\n"),
			true,
		},
		{"file declares an empty hosts list", []byte("hosts: []\n"), true},
		{"unparsable file", []byte("not: [valid: yaml"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileDeclaresHosts(tt.fileBytes); got != tt.want {
				t.Errorf("fileDeclaresHosts(%q) = %v, want %v", tt.fileBytes, got, tt.want)
			}
		})
	}
}

func TestDiscoveredConfigHosts(t *testing.T) {
	auths := []github.HostAuth{
		{Name: "github.com", GitProtocol: "ssh"},
		{Name: "ghe.corp.internal", GitProtocol: ""}, // gh's file didn't record one
	}

	got := discoveredConfigHosts(auths, "/src")

	want := []config.HostConfig{
		{Name: "github.com", Forge: "github", Frontdoor: "gh-cli", Protocol: "ssh", DefaultTarget: "/src"},
		{Name: "ghe.corp.internal", Forge: "github", Frontdoor: "gh-cli", Protocol: "ssh", DefaultTarget: "/src"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d hosts, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("host[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestResolveHosts(t *testing.T) {
	ghe := config.HostConfig{Name: "ghe.corp", Forge: "github", Protocol: "ssh"}
	gh := config.HostConfig{Name: "github.com", Forge: "github", Protocol: "ssh"}
	gheHTTPS := config.HostConfig{Name: "ghe.corp", Forge: "github", Protocol: "https"}
	bb := config.HostConfig{Name: "bitbucket.org", Forge: "bitbucket", Protocol: "https"}

	tests := []struct {
		name           string
		discovered     []config.HostConfig
		file           []config.HostConfig
		declared       bool
		want           []config.HostConfig
		wantConfigured []string
	}{
		{"no hosts key uses discovery", []config.HostConfig{ghe}, []config.HostConfig{gh}, false, []config.HostConfig{ghe}, nil},
		{"no hosts key, nothing discovered keeps the implicit default", nil, []config.HostConfig{gh}, false, []config.HostConfig{gh}, nil},
		{"file hosts follow discovered ones", []config.HostConfig{ghe}, []config.HostConfig{bb}, true,
			[]config.HostConfig{ghe, bb}, []string{"bitbucket.org"}},
		{"a file entry overrides a discovered host in place", []config.HostConfig{gh, ghe}, []config.HostConfig{bb, gheHTTPS}, true,
			[]config.HostConfig{gh, gheHTTPS, bb}, []string{"bitbucket.org", "ghe.corp"}},
		{"discovery off (nil) is exactly the file list", nil, []config.HostConfig{bb}, true, []config.HostConfig{bb}, []string{"bitbucket.org"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, configured := resolveHosts(tt.discovered, tt.file, tt.declared)
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Errorf("hosts = %v, want %v", got, tt.want)
			}
			if len(configured) != len(tt.wantConfigured) {
				t.Errorf("configured = %v, want %v", configured, tt.wantConfigured)
			}
			for _, n := range tt.wantConfigured {
				if !configured[n] {
					t.Errorf("configured = %v, missing %s", configured, n)
				}
			}
		})
	}
}
