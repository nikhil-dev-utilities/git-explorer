package glyphclone

import (
	"io"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

// Hooks connect an embedded Screen to its host Glyph app.
type Hooks struct {
	Spawn   func(func())          // run off the render goroutine
	Apply   func(func())          // queue onto the render goroutine
	Refresh func()                // request a frame
	Push    func(*riffkey.Router) // push a modal key router
	Pop     func()                // pop the top router
	Done    func(Response)        // the user left the screen
}

// Embedded is the clone dialog and run, mounted inside a larger Glyph app. It is created
// once; Open starts it for a Request. The host shows View() while the screen is open
// and pushes Router() as the modal key router. The run log is one continuous stream
// for the life of the Embedded screen (Glyph's Log cannot be cleared), with a header line per run.
type Embedded struct {
	s      *state
	view   Component
	router *riffkey.Router
}

func Embed(h Hooks) *Embedded {
	pr, pw := io.Pipe()
	s := &state{logw: pw, spawn: h.Spawn, apply: h.Apply, refresh: h.Refresh}
	s.quit = func() { h.Done(s.output()) }

	prompt := promptRouter(s)
	s.enterPrompt = func() { h.Push(prompt) }
	s.leavePrompt = h.Pop

	router := riffkey.NewRouter()
	bindKeys(s, func(pattern string, fn func()) {
		router.Handle(pattern, func(riffkey.Match) { fn(); h.Refresh() })
	})
	return &Embedded{
		s:      s,
		view:   VBox.Grow(1)(If(&s.showRun).Then(runView(s, pr)).Else(dialogView(s))),
		router: router,
	}
}

// Open (re)initialises the screen for req. The host makes it visible.
func (sc *Embedded) Open(req Request) { sc.s.reset(req) }

func (sc *Embedded) View() Component { return sc.view }

func (sc *Embedded) Router() *riffkey.Router { return sc.router }
