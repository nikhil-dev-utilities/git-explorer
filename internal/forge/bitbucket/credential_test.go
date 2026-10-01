package bitbucket

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

const secret = "s3cret-token"

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func fillReturning(cred credential, ok bool, err error) credentialFillFunc {
	return func(context.Context, string) (credential, bool, error) { return cred, ok, err }
}

func fillMustNotRun(t *testing.T) credentialFillFunc {
	return func(context.Context, string) (credential, bool, error) {
		t.Error("git credential fill ran, want the environment to win")
		return credential{}, false, nil
	}
}

// wantFatalWithoutSecret checks err is Fatal and that the token appears nowhere in it.
func wantFatalWithoutSecret(t *testing.T, err error) {
	t.Helper()
	var fErr *forge.Error
	if !errors.As(err, &fErr) || fErr.Kind != forge.ErrKindFatal {
		t.Fatalf("error = %v, want a Fatal *forge.Error", err)
	}
	for e := error(fErr); e != nil; e = errors.Unwrap(e) {
		if strings.Contains(e.Error(), secret) {
			t.Errorf("error text %q contains the token", e.Error())
		}
	}
}

func TestResolveCredential(t *testing.T) {
	ctx := context.Background()
	stored := credential{username: "me@example.com", password: secret}

	t.Run("environment wins over the store", func(t *testing.T) {
		got, err := resolveCredential(ctx, env(map[string]string{"BITBUCKET_EMAIL": "env@example.com", "BITBUCKET_API_TOKEN": secret}),
			fillMustNotRun(t), "api.bitbucket.org")
		if err != nil || got.username != "env@example.com" || got.password != secret {
			t.Errorf("got %+v, %v", got.username, err)
		}
	})
	t.Run("only one environment variable is an error", func(t *testing.T) {
		_, err := resolveCredential(ctx, env(map[string]string{"BITBUCKET_API_TOKEN": secret}), fillMustNotRun(t), "api.bitbucket.org")
		wantFatalWithoutSecret(t, err)
	})
	t.Run("store hit", func(t *testing.T) {
		got, err := resolveCredential(ctx, env(nil), fillReturning(stored, true, nil), "api.bitbucket.org")
		if err != nil || got != stored {
			t.Errorf("got %v, %v; want the stored credential", got.username, err)
		}
	})
	t.Run("store miss is Fatal", func(t *testing.T) {
		_, err := resolveCredential(ctx, env(nil), fillReturning(credential{}, false, nil), "api.bitbucket.org")
		wantFatalWithoutSecret(t, err)
	})
	t.Run("store failure is Fatal", func(t *testing.T) {
		_, err := resolveCredential(ctx, env(nil), fillReturning(credential{}, false, context.DeadlineExceeded), "api.bitbucket.org")
		wantFatalWithoutSecret(t, err)
	})
}

// isolateGit points git at an empty global config in a temp dir, with an optional
// credential helper line, so the test never touches the developer's keychain.
func isolateGit(t *testing.T, helper string) {
	t.Helper()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "gitconfig")
	content := ""
	if helper != "" {
		content = "[credential]\n\thelper = " + helper + "\n"
	}
	if err := os.WriteFile(cfg, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("HOME", dir)
}

func TestGitCredentialFill_ReadsTheStore(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(dir, "creds")
	if err := os.WriteFile(store, []byte("https://me%40example.com:"+secret+"@api.bitbucket.org\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	isolateGit(t, "store --file "+store)

	got, ok, err := gitCredentialFill(context.Background(), "api.bitbucket.org")
	if err != nil || !ok || got.username != "me@example.com" || got.password != secret {
		t.Errorf("gitCredentialFill = %v, %v, %v; want the stored credential", got.username, ok, err)
	}
}

func TestGitCredentialFill_MissingEntryFailsFastWithoutPrompting(t *testing.T) {
	isolateGit(t, "")

	_, ok, err := gitCredentialFill(context.Background(), "api.bitbucket.org")
	if ok || err != nil {
		t.Errorf("gitCredentialFill = ok %v, err %v; want a plain miss", ok, err)
	}
}
