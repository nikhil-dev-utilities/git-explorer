package github

import (
	"context"
	"fmt"
	"strings"

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
		return nil, &forge.Error{
			Kind:    forge.ErrKindPaneScoped,
			Message: fmt.Sprintf("gh api %s failed", endpoint),
			Err:     fmt.Errorf("gh api %s: %s", endpoint, strings.TrimSpace(string(res.Stderr))),
		}
	}
	return res.Stdout, nil
}
