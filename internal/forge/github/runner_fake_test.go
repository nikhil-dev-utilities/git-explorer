package github

import (
	"context"
	"fmt"
	"strings"
)

// fakeRunner is the seam every test in this package injects instead of execRunner. It
// matches on the exact argument list, which is fine here: the adapter's argument
// construction is entirely under this package's control, so tests can assert the
// precise gh invocation they expect.
type fakeRunner struct {
	responses map[string]fakeResponse
	calls     [][]string
}

type fakeResponse struct {
	result runResult
	err    error
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{responses: make(map[string]fakeResponse)}
}

// on registers a successful (from the process's point of view — ExitCode may still be
// non-zero) response for the exact given args.
func (f *fakeRunner) on(args []string, res runResult) {
	f.responses[argKey(args)] = fakeResponse{result: res}
}

// onStartFailure registers a "could not start the process at all" response, the shape
// execRunner returns when gh isn't installed.
func (f *fakeRunner) onStartFailure(args []string, err error) {
	f.responses[argKey(args)] = fakeResponse{err: err}
}

func (f *fakeRunner) Run(_ context.Context, args ...string) (runResult, error) {
	f.calls = append(f.calls, append([]string{}, args...))

	resp, ok := f.responses[argKey(args)]
	if !ok {
		return runResult{}, fmt.Errorf("fakeRunner: no response configured for args %v", args)
	}
	return resp.result, resp.err
}

func argKey(args []string) string {
	return strings.Join(args, "\x1f")
}
