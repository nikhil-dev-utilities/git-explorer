package main

import (
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
		{Name: "github.com", Frontdoor: "gh-cli", Protocol: "ssh", DefaultTarget: "/src"},
		{Name: "ghe.corp.internal", Frontdoor: "gh-cli", Protocol: "ssh", DefaultTarget: "/src"},
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
