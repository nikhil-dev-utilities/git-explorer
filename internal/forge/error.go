package forge

import "time"

// ErrorKind classifies a Forge error into one of the three failure surfaces the TUI
// renders differently, matched to blast radius (see DESIGN.md, "Failure surfaces").
type ErrorKind int

const (
	// ErrKindFatal means nothing else in the app works until this is fixed: gh isn't
	// installed, or the active Host isn't authenticated.
	ErrKindFatal ErrorKind = iota
	// ErrKindTransient resolves itself by waiting — a rate limit.
	ErrKindTransient
	// ErrKindPaneScoped affects one pane's data, not the whole app. Data already
	// loaded before this error must not be discarded because of it.
	ErrKindPaneScoped
)

// Error is the error type every Forge implementation returns, so callers can classify
// failures without depending on any implementation's internals.
type Error struct {
	Kind ErrorKind
	// Message is human-readable and safe to display. It must never contain credential
	// material — see ADR-0004.
	Message string
	// RetryAfter is meaningful only when Kind is ErrKindTransient.
	RetryAfter time.Duration
	// Err is the wrapped underlying cause, if any.
	Err error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}
