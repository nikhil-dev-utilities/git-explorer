package github

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// checkAuth is a proactive auth-check probe, run before any data-fetching call for a
// Host. It asks gh for a credential and then discards it immediately — the token bytes
// are never stored, logged, or returned beyond this function, and are used only to
// observe whether the call succeeded. See ADR-0004. Its own log lines name the Host
// and outcome only, never the credential.
//
// This exists so an unauthenticated Host fails fast with a clear, specific message,
// rather than surfacing as a generic failure from whichever data call happens to run
// first.
func (a *Adapter) checkAuth(ctx context.Context, host forge.Host) error {
	res, runErr := a.run.Run(ctx, "auth", "token", "--hostname", host.Name)
	if runErr != nil {
		err := notInstalledError(runErr)
		slog.ErrorContext(ctx, "gh not installed", "host", host.Name, "error", err)
		return err
	}
	if res.ExitCode != 0 {
		slog.WarnContext(ctx, "not authenticated", "host", host.Name)
		return notAuthenticatedError(host, res.Stderr)
	}
	// res.Stdout holds the credential here. It is deliberately never read.
	slog.DebugContext(ctx, "authenticated", "host", host.Name)
	return nil
}

func notInstalledError(cause error) error {
	return &forge.Error{
		Kind:    forge.ErrKindFatal,
		Message: "gh is not installed or not on PATH. Install it from https://cli.github.com and try again.",
		Err:     cause,
	}
}

func notAuthenticatedError(host forge.Host, stderr []byte) error {
	return &forge.Error{
		Kind: forge.ErrKindFatal,
		Message: fmt.Sprintf(
			"not authenticated for %s. Run: gh auth login --hostname %s",
			host.Name, host.Name,
		),
		Err: fmt.Errorf("gh auth token: %s", strings.TrimSpace(string(stderr))),
	}
}
