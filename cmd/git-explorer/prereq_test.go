package main

import (
	"errors"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/config"
	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// onPath fakes exec.LookPath with only the named binaries present.
func onPath(names ...string) lookPathFunc {
	return func(file string) (string, error) {
		for _, n := range names {
			if n == file {
				return "/usr/bin/" + file, nil
			}
		}
		return "", errors.New("not found")
	}
}

func wantFatal(t *testing.T, err error) {
	t.Helper()
	var fErr *forge.Error
	if !errors.As(err, &fErr) || fErr.Kind != forge.ErrKindFatal {
		t.Errorf("error = %v, want a Fatal *forge.Error", err)
	}
}

func TestCheckGit(t *testing.T) {
	if err := checkGit(onPath("git")); err != nil {
		t.Errorf("checkGit with git present = %v, want nil", err)
	}
	wantFatal(t, checkGit(onPath()))
}

func TestCheckGh(t *testing.T) {
	github := []config.HostConfig{{Name: "github.com", Forge: "github"}}
	bitbucket := []config.HostConfig{{Name: "bitbucket.org", Forge: "bitbucket"}}

	if err := checkGh(bitbucket, onPath()); err != nil {
		t.Errorf("Bitbucket-only Hosts without gh = %v, want nil", err)
	}
	if err := checkGh(github, onPath("gh")); err != nil {
		t.Errorf("GitHub Host with gh = %v, want nil", err)
	}
	wantFatal(t, checkGh(append(bitbucket, github...), onPath()))
}
