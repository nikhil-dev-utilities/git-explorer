package config

import "os"

// Environ abstracts environment variable lookup so Load's core logic never calls
// os.Getenv directly — tests inject a synthetic Environ instead of mutating the real
// process environment.
type Environ interface {
	Getenv(key string) string
}

// OSEnviron reads from the real process environment. Use it in production; tests use
// MapEnviron instead.
type OSEnviron struct{}

func (OSEnviron) Getenv(key string) string {
	return os.Getenv(key)
}

// MapEnviron is a synthetic Environ backed by a plain map, for tests.
type MapEnviron map[string]string

func (m MapEnviron) Getenv(key string) string {
	return m[key]
}
