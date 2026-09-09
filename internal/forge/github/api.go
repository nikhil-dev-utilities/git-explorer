package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// runAPI calls `gh api --hostname <host> -X GET <endpoint> [extraArgs...]` and returns
// its raw stdout on success. gh handles authentication for the call internally using
// its own credential store — this function never sees a token.
//
// -X GET is explicit and non-negotiable: gh api defaults to POST whenever any -f/-F
// flag is present (used throughout this package for pagination/filter params like
// per_page), and every endpoint this package calls is a read-only listing endpoint
// with no POST route — silently POSTing to one 404s. Every call in this package goes
// through this one function, so forcing GET here fixes all of them at once. See
// https://github.com/nikhil-dev-utilities/git-explorer/issues/57.
//
// Every call is logged through slog.Default() — endpoint and outcome only, never
// stdout/stderr content, which could carry response data (never credentials: gh
// itself owns the token, this package never sees one). This is the single seam
// every gh api call in this package goes through, so logging here covers all of
// them without scattering call sites — see
// https://github.com/nikhil-dev-utilities/git-explorer/issues/80.
func (a *Adapter) runAPI(ctx context.Context, host forge.Host, endpoint string, extraArgs ...string) ([]byte, error) {
	args := append([]string{"api", "--hostname", host.Name, "-X", "GET", endpoint}, extraArgs...)

	res, runErr := a.run.Run(ctx, args...)
	if runErr != nil {
		err := notInstalledError(runErr)
		slog.ErrorContext(ctx, "gh api call could not start", "host", host.Name, "endpoint", endpoint, "error", err)
		return nil, err
	}
	if res.ExitCode != 0 {
		err := classifyAPIFailure(endpoint, res.Stderr)
		slog.WarnContext(ctx, "gh api call failed", "host", host.Name, "endpoint", endpoint, "kind", err.Kind, "message", err.Message)
		return nil, err
	}
	slog.DebugContext(ctx, "gh api call succeeded", "host", host.Name, "endpoint", endpoint, "response_bytes", len(res.Stdout))
	return res.Stdout, nil
}

const (
	rateLimitSignal            = "API rate limit exceeded"
	defaultRateLimitRetryAfter = 60 * time.Second
)

var retryAfterPattern = regexp.MustCompile(`(?i)retry-after:\s*(\d+)`)

// classifyAPIFailure turns a non-zero gh api exit into a forge.Error. A rate limit is
// classified transient and carries a retry-after duration; anything else is
// pane-scoped, distinct from the fatal not-authenticated/not-installed errors auth.go
// owns.
func classifyAPIFailure(endpoint string, stderr []byte) *forge.Error {
	text := strings.TrimSpace(string(stderr))

	if strings.Contains(text, rateLimitSignal) {
		return &forge.Error{
			Kind:       forge.ErrKindTransient,
			Message:    fmt.Sprintf("rate limited by GitHub while calling %s", endpoint),
			RetryAfter: parseRetryAfter(text),
			Err:        errors.New(text),
		}
	}

	return &forge.Error{
		Kind:    forge.ErrKindPaneScoped,
		Message: fmt.Sprintf("gh api %s failed", endpoint),
		Err:     errors.New(text),
	}
}

// parseRetryAfter looks for a "Retry-After: <seconds>" hint in gh's error text. gh does
// not reliably surface GitHub's Retry-After response header through its CLI error
// output, so when no explicit duration is found this falls back to a conservative fixed
// wait rather than leaving RetryAfter unset.
func parseRetryAfter(text string) time.Duration {
	if m := retryAfterPattern.FindStringSubmatch(text); m != nil {
		if secs, err := strconv.Atoi(m[1]); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultRateLimitRetryAfter
}
