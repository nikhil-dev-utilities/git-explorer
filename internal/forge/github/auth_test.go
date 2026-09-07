package github

import (
	"context"
	"strings"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func TestCheckAuth_Success(t *testing.T) {
	host := forge.Host{Name: "github.com"}
	fr := newFakeRunner()
	// A canary credential in stdout. checkAuth's signature returns only an error, so
	// the only way this value could leak is if a future change starts returning or
	// logging it — this test pins the current, correct behavior: it's discarded.
	fr.on([]string{"auth", "token", "--hostname", "github.com"}, runResult{
		Stdout:   []byte("canary-token-must-never-leak"),
		ExitCode: 0,
	})

	a := newWithRunner(fr)
	if err := a.checkAuth(context.Background(), host); err != nil {
		t.Fatalf("checkAuth() = %v, want nil", err)
	}
}

func TestCheckAuth_NotAuthenticated(t *testing.T) {
	host := forge.Host{Name: "ghe.corp.internal"}
	fr := newFakeRunner()
	fr.on([]string{"auth", "token", "--hostname", "ghe.corp.internal"}, runResult{
		Stderr:   []byte("You are not logged into any GitHub hosts.\n"),
		ExitCode: 1,
	})

	a := newWithRunner(fr)
	err := a.checkAuth(context.Background(), host)
	if err == nil {
		t.Fatal("checkAuth() = nil, want an error")
	}

	fe, ok := err.(*forge.Error)
	if !ok {
		t.Fatalf("error = %v (%T), want a *forge.Error", err, err)
	}
	if fe.Kind != forge.ErrKindFatal {
		t.Fatalf("Kind = %v, want ErrKindFatal", fe.Kind)
	}
	if !strings.Contains(fe.Message, "gh auth login --hostname ghe.corp.internal") {
		t.Fatalf("Message = %q, want it to name the fix command", fe.Message)
	}
}

func TestCheckAuth_GhNotInstalled(t *testing.T) {
	host := forge.Host{Name: "github.com"}
	fr := newFakeRunner()
	fr.onStartFailure([]string{"auth", "token", "--hostname", "github.com"}, errGhNotFound)

	a := newWithRunner(fr)
	err := a.checkAuth(context.Background(), host)
	if err == nil {
		t.Fatal("checkAuth() = nil, want an error")
	}

	fe, ok := err.(*forge.Error)
	if !ok {
		t.Fatalf("error = %v (%T), want a *forge.Error", err, err)
	}
	if fe.Kind != forge.ErrKindFatal {
		t.Fatalf("Kind = %v, want ErrKindFatal", fe.Kind)
	}
}
