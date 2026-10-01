// Package bitbucket is the Bitbucket Cloud Forge adapter. It reaches Bitbucket through
// one private Frontdoor, its REST API (ADR-0001, ADR-0010).
package bitbucket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// credential is a Basic-auth pair for the REST API: the Atlassian account email and an
// API token. It is held in memory only and must never be logged or put in an error.
type credential struct {
	username string
	password string
}

// credentialFillFunc asks git's credential store for host's credential. ok is false
// when the store has no entry. It is the seam tests replace.
type credentialFillFunc func(ctx context.Context, host string) (cred credential, ok bool, err error)

// resolveCredential finds the credential for apiHost, in the order ADR-0011 fixes:
// BITBUCKET_EMAIL + BITBUCKET_API_TOKEN from the environment, then git's credential
// store. Every failure is Fatal: nothing on this Host works without a credential.
func resolveCredential(ctx context.Context, getenv func(string) string, fill credentialFillFunc, apiHost string) (credential, error) {
	email, token := getenv("BITBUCKET_EMAIL"), getenv("BITBUCKET_API_TOKEN")
	switch {
	case email != "" && token != "":
		slog.DebugContext(ctx, "bitbucket credential from environment", "host", apiHost)
		return credential{username: email, password: token}, nil
	case email != "" || token != "":
		return credential{}, &forge.Error{
			Kind:    forge.ErrKindFatal,
			Message: "set both BITBUCKET_EMAIL and BITBUCKET_API_TOKEN, or neither.",
		}
	}

	cred, ok, err := fill(ctx, apiHost)
	if err != nil {
		slog.WarnContext(ctx, "git credential fill failed", "host", apiHost, "error", err)
	}
	if err != nil || !ok {
		return credential{}, notAuthenticatedError(apiHost, err)
	}
	slog.DebugContext(ctx, "bitbucket credential from git credential store", "host", apiHost)
	return cred, nil
}

func notAuthenticatedError(apiHost string, cause error) error {
	return &forge.Error{
		Kind: forge.ErrKindFatal,
		Message: fmt.Sprintf(
			"not authenticated for %s. Set BITBUCKET_EMAIL and BITBUCKET_API_TOKEN, or store them in git's credential helper: "+
				"printf 'protocol=https\\nhost=%s\\nusername=<email>\\npassword=<api token>\\n\\n' | git credential approve",
			apiHost, apiHost),
		Err: cause,
	}
}

// fillTimeout bounds git credential fill, so a helper that hangs cannot hang the UI.
const fillTimeout = 10 * time.Second

// gitCredentialFill is the real credentialFillFunc. It runs `git credential fill`
// with every prompt disabled, so a missing entry fails fast instead of asking:
// GIT_TERMINAL_PROMPT=0 stops terminal prompts, and GIT_ASKPASS=true (which outranks
// core.askPass and SSH_ASKPASS) answers any GUI prompt with an empty string.
//
// Its stdout is the credential itself (ADR-0011): it is parsed here and never logged,
// and neither stdout nor stderr is ever put in a returned error.
func gitCredentialFill(ctx context.Context, host string) (credential, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, fillTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "credential", "fill")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=" + host + "\n\n")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=true", "GCM_INTERACTIVE=never")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && ctx.Err() == nil {
			// git exits non-zero when it would have had to prompt: no entry.
			return credential{}, false, nil
		}
		if ctx.Err() != nil {
			return credential{}, false, fmt.Errorf("git credential fill: %w", ctx.Err())
		}
		return credential{}, false, fmt.Errorf("git credential fill: %w", err)
	}

	cred := parseCredential(stdout.Bytes())
	return cred, cred.username != "" && cred.password != "", nil
}

// parseCredential reads git's key=value credential output (see git-credential(1)).
func parseCredential(out []byte) credential {
	var c credential
	for _, line := range strings.Split(string(out), "\n") {
		key, value, _ := strings.Cut(line, "=")
		switch key {
		case "username":
			c.username = value
		case "password":
			c.password = value
		}
	}
	return c
}
