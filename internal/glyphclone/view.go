package glyphclone

import (
	"io"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

func dialogView(s *state) Component {
	return VBox.Grow(1)(
		HBox.Gap(1)(Text(&s.title).Bold(), Text("→").Dim(), Text(&s.dirDisplay).Bold().FG(Cyan)),
		HBox.Grow(1)(
			VBox.WidthPct(0.4).Border(BorderRounded).Title("Choose target folder")(
				List(&s.entries).Selection(&s.cursor).
					Render(func(e *entry) Component { return Text(&e.Show) }),
			),
			VBox.Grow(1).Border(BorderRounded).Title("Preview")(
				TextView(&s.previewText).Grow(1),
			),
		),
		If(&s.prompting).
			Then(HBox.Gap(1)(Text(&s.promptLabel).Bold(), Input().Field(&s.field).Width(70))).
			Else(Text(&s.options)),
		Text(&s.notice).FG(Red),
		Text(&s.hint).Dim(),
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

// bindKeys registers the dialog and run keys through bind. List navigation is ours (s.move) rather than the
// List's BindNav, which would register j/k/g/G on the host app's base router.
func bindKeys(s *state, bind func(pattern string, fn func())) {
	for _, k := range []string{"j", "<Down>"} {
		bind(k, func() { s.move(1) })
	}
	for _, k := range []string{"k", "<Up>"} {
		bind(k, func() { s.move(-1) })
	}
	bind("g", func() { s.jump(false) })
	bind("G", func() { s.jump(true) })
	for _, k := range []string{"<Enter>", "l", "<Right>"} {
		bind(k, s.open)
	}
	for _, k := range []string{"h", "<Left>", "<BS>"} {
		bind(k, s.up)
	}
	bind("/", s.openGoto)
	bind("n", s.openNew)
	bind("<Tab>", s.toggleOrgSubdir)
	bind("c", s.confirm)
	bind("r", s.retry)
	bind("<Esc>", s.back)
	bind("<C-c>", s.back)
}

// promptRouter owns every key while the path or new-folder prompt is open.
func promptRouter(s *state) *riffkey.Router {
	// NoCounts: otherwise riffkey swallows digits as vim count prefixes and a path
	// like ~/src2026 cannot be typed.
	prompt := riffkey.NewRouter().NoCounts()
	prompt.Handle("<Enter>", func(riffkey.Match) { s.submitPrompt() })
	prompt.Handle("<Esc>", func(riffkey.Match) { s.back() })
	prompt.Handle("<C-c>", func(riffkey.Match) { s.back() })
	prompt.TextInput(&s.field.Value, &s.field.Cursor)
	return prompt
}
