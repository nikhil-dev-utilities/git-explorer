package config

import (
	"strings"
	"testing"
)

func TestLoad_HostForgeAndFrontdoorDefaults(t *testing.T) {
	yaml := `
hosts:
  - name: github.com
  - name: bitbucket.org
    forge: bitbucket
`
	cfg, err := Load(Flags{}, MapEnviron{"HOME": "/home/nikhil"}, []byte(yaml))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	gh, bb := cfg.Hosts[0], cfg.Hosts[1]
	if gh.Forge != "github" || gh.Frontdoor != "gh-cli" {
		t.Errorf("github.com = %+v, want forge github, frontdoor gh-cli", gh)
	}
	if bb.Forge != "bitbucket" || bb.Frontdoor != "rest" {
		t.Errorf("bitbucket.org = %+v, want forge bitbucket, frontdoor rest", bb)
	}
}

func TestLoad_RejectsBadForgeOrFrontdoor(t *testing.T) {
	tests := []struct {
		name, yaml, wantErr string
	}{
		{"unknown forge", "hosts:\n  - name: x\n    forge: gitlab\n", `unknown forge "gitlab"`},
		{"frontdoor from another forge", "hosts:\n  - name: x\n    forge: bitbucket\n    frontdoor: gh-cli\n",
			`frontdoor "gh-cli" does not belong to forge bitbucket`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(Flags{}, MapEnviron{"HOME": "/home/nikhil"}, []byte(tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
