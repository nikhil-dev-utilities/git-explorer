package github

import (
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestCloneURL(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		want     string
	}{
		{name: "ssh", protocol: "ssh", want: "git@github.com:acme/api.git"},
		{name: "https", protocol: "https", want: "https://github.com/acme/api.git"},
		{name: "unset defaults to ssh", protocol: "", want: "git@github.com:acme/api.git"},
	}

	a := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := forge.Repo{
				Name: "api",
				Org:  "acme",
				Host: forge.Host{Name: "github.com", Protocol: tt.protocol},
			}
			if got := a.CloneURL(repo); got != tt.want {
				t.Errorf("CloneURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCloneURL_PrivateHost(t *testing.T) {
	a := New()
	repo := forge.Repo{
		Name: "infra",
		Org:  "platform",
		Host: forge.Host{Name: "ghe.corp.internal", Protocol: "https"},
	}
	want := "https://ghe.corp.internal/platform/infra.git"
	if got := a.CloneURL(repo); got != want {
		t.Errorf("CloneURL() = %q, want %q", got, want)
	}
}
