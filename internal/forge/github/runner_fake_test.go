package github

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// fakeRunner is the seam every test in this package injects instead of execRunner. It
// matches on the exact argument list, which is fine here: the adapter's argument
// construction is entirely under this package's control, so tests can assert the
// precise gh invocation they expect.
//
// It is safe for concurrent use: some tests exercise ListOrgs's Private Host path,
// which fetches pages from a goroutine while the test goroutine reads the resulting
// channel and inspects calls.
type fakeRunner struct {
	mu        sync.Mutex
	responses map[string]fakeResponse
	calls     [][]string
}

type fakeResponse struct {
	result runResult
	err    error
	// gate, if set, is received from before the call is recorded or its response
	// returned. This lets a test prove a later call has not happened yet
	// deterministically — by controlling exactly when the gate opens — rather than
	// racing on timing.
	gate <-chan struct{}
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{responses: make(map[string]fakeResponse)}
}

// on registers a successful (from the process's point of view — ExitCode may still be
// non-zero) response for the exact given args.
func (f *fakeRunner) on(args []string, res runResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses[argKey(args)] = fakeResponse{result: res}
}

// onStartFailure registers a "could not start the process at all" response, the shape
// execRunner returns when gh isn't installed.
func (f *fakeRunner) onStartFailure(args []string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses[argKey(args)] = fakeResponse{err: err}
}

// onGated registers a response that is only recorded and returned once gate is closed.
func (f *fakeRunner) onGated(args []string, res runResult, gate <-chan struct{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses[argKey(args)] = fakeResponse{result: res, gate: gate}
}

func (f *fakeRunner) Run(_ context.Context, args ...string) (runResult, error) {
	f.mu.Lock()
	resp, ok := f.responses[argKey(args)]
	f.mu.Unlock()

	if !ok {
		f.record(args)
		return runResult{}, fmt.Errorf("fakeRunner: no response configured for args %v", args)
	}
	if resp.gate != nil {
		<-resp.gate
	}
	f.record(args)
	return resp.result, resp.err
}

func (f *fakeRunner) record(args []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, append([]string{}, args...))
}

// snapshotCalls returns a copy of the calls made so far, safe to read while another
// goroutine may still be calling Run.
func (f *fakeRunner) snapshotCalls() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string{}, f.calls...)
}

func argKey(args []string) string {
	return strings.Join(args, "\x1f")
}
