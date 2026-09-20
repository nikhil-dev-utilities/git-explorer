package glyphclone

import (
	"io"
	"os"

	. "github.com/kungfusheep/glyph"
)

const dialogHints = "enter open · ←/backspace up · tab org subdirectory · c clone · esc cancel"

func dialogView(s *state) Component {
	return VBox.Grow(1)(
		HBox.Gap(1)(Text(&s.title).Bold(), Text("→").Dim(), Text(&s.dirDisplay).Bold().FG(Cyan)),
		HBox.Grow(1)(
			VBox.WidthPct(0.4).Border(BorderRounded).Title("Choose target folder")(
				List(&s.entries).Selection(&s.cursor).
					Render(func(e *entry) Component { return Text(&e.Show) }).
					BindNav("j", "k").BindNav("<Down>", "<Up>").BindFirstLast("g", "G"),
			),
			VBox.Grow(1).Border(BorderRounded).Title("Preview")(
				TextView(&s.previewText).Grow(1),
			),
		),
		Text(&s.options),
		Text(dialogHints).Dim(),
	)
}

func runView(s *state, logr io.Reader) Component {
	return VBox.Grow(1)(
		HBox.Gap(2)(If(&s.busy).Then(Spinner().FG(Cyan)), Text(&s.status).Bold(), Progress(&s.pct).Width(20)),
		VBox.Grow(1).Border(BorderRounded).Title("Clone log")(
			Log(logr).Grow(1).MaxLines(2000),
		),
		Text(&s.keys).Dim(),
	)
}

// Launch runs the clone screen to completion and blocks until the user leaves it. It
// takes over the terminal, so a bubbletea shell must release it first (tea.Exec).
func Launch(in Request) (Response, error) {
	// Glyph's App.Stop closes os.Stdin. Give it a private /dev/tty handle so the
	// bubbletea shell's own stdin survives this screen and can resume afterwards.
	// wake (a second handle, since Stop closes the first) unblocks Glyph's pending
	// read on Stop; see wakeRead.
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return Response{}, err
	}
	defer tty.Close()
	waker, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return Response{}, err
	}
	defer waker.Close()
	orig := os.Stdin
	os.Stdin = tty
	defer func() { os.Stdin = orig }()

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	app := NewApp()
	stop := func() {
		app.Stop()
		wakeRead(waker)
	}
	s := newState(in, pw, func(f func()) { go f() }, app.Apply, app.RequestRender, stop)
	wire(app, s, pr)

	err = app.Run()
	return s.output(), err
}

// wire uses one view with the dialog/run switch inside it, rather than named views:
// Glyph does not expose a named view's template, which would make it untestable
// headlessly. Every handler is a no-op outside its own phase (see state).
func wire(app *App, s *state, logr io.Reader) {
	app.SetView(VBox.Grow(1)(If(&s.showRun).Then(runView(s, logr)).Else(dialogView(s)))).
		Handle("<Enter>", s.open).
		Handle("l", s.open).
		Handle("<Right>", s.open).
		Handle("h", s.up).
		Handle("<Left>", s.up).
		Handle("<BS>", s.up).
		Handle("<Tab>", s.toggleOrgSubdir).
		Handle("c", s.confirm).
		Handle("r", s.retry).
		Handle("<Esc>", s.back).
		Handle("<C-c>", s.back)
}
