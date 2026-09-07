package github

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// runAPI calls `gh api --hostname <host> <endpoint> [extraArgs...]` and returns its raw
// stdout on success. gh handles authentication for the call internally using its own
// credential store — this function never sees a token.
func (a *Adapter) runAPI(ctx context.Context, host forge.Host, endpoint string, extraArgs ...string) ([]byte, error) {
	args := append([]string{"api", "--hostname", host.Name, endpoint}, extraArgs...)

	res, runErr := a.run.Run(ctx, args...)
	if runErr != nil {
		return nil, notInstalledError(runErr)
	}
	if res.ExitCode != 0 {
		return nil, classifyAPIFailure(endpoint, res.Stderr)
	}
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
func classifyAPIFailure(endpoint string, stderr []byte) error {
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
