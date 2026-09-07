// Package github implements forge.Forge for GitHub (github.com and self-managed
// enterprise installs), reached through the gh CLI Frontdoor. See ADR-0001.
package github

// Adapter implements forge.Forge. Construct it with New; newWithRunner exists for this
// package's own tests to inject a fake runner.
type Adapter struct {
	run runner
}

// New returns an Adapter backed by the real gh binary on PATH.
func New() *Adapter {
	return &Adapter{run: execRunner{}}
}

func newWithRunner(r runner) *Adapter {
	return &Adapter{run: r}
}
