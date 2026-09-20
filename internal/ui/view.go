package ui

import (
	. "github.com/kungfusheep/glyph"
)

var modalFill = RGB(0x1c, 0x1c, 0x1c)

func orgPane(s *state) Component {
	s.orgFL.Render(func(r *orgRow) Component {
		return HBox.Gap(1)(Text(&r.Org.Name), Space(), Text(&r.Aff).Dim())
	})
	return VBox.Border(BorderRounded).BorderFG(&s.orgBorder).Width(&s.orgW).Height(&s.paneH)(
		Text(&s.orgTitle).Bold(),
		Text(&s.orgChips).Dim(),
		s.orgFL,
		Text(&s.orgMsg).FG(Yellow),
	)
}

func repoPane(s *state) Component {
	s.repoFL.Render(func(r *repoRow) Component {
		return HBox(
			If(&r.Ticked).Then(Text("[x]").FG(Green)).Else(Text("[ ]").Dim()),
			SpaceW(1),
			Text(&r.Repo.Name),
			Space(),
			If(&s.lay.showBadges).Then(Text(&r.Badges).FG(Yellow)),
			If(&s.lay.showAge).Then(HBox(SpaceW(1), Text(&r.Age).Dim())),
		)
	})
	return VBox.Border(BorderRounded).BorderFG(&s.repoBorder).Grow(1).Height(&s.paneH)(
		Text(&s.repoTitle).Bold(),
		Text(&s.repoChips).Dim(),
		s.repoFL,
		Text(&s.repoMsg).FG(Yellow),
	)
}

func browseView(s *state) Component {
	return VBox.Grow(1)(
		HBox.Height(&s.paneH)(orgPane(s), repoPane(s)),
		Text(&s.footer).Bold(),
		Text(&s.status).FG(Yellow),
		Text(&s.hint).Dim(),
	)
}

func fatalView(s *state) Component {
	return VBox.Grow(1)(
		Text("git-explorer cannot continue").Bold().FG(Red),
		Text(""),
		Text(&s.fatalText),
		Text(""),
		Text("^y switch host · ^c quit").Dim(),
	)
}

// rootView keeps the panes outside any conditional: Glyph wires a conditional
// branch's key bindings through child scopes, which would sit ahead of the routing
// override in wire. Full-screen states are overlays drawn over the panes instead.
func rootView(s *state) Component {
	return VBox.Grow(1)(
		browseView(s),
		If(&s.showFatal).Then(Overlay.Backdrop().Centered()(VBox.Border(BorderRounded).Fill(modalFill).Padding(1).FitContent()(fatalView(s)))),
		If(&s.showTooNarrow).Then(Overlay.Backdrop().Centered()(VBox.Border(BorderRounded).Fill(modalFill).Padding(1).FitContent()(Text("terminal too narrow: widen to 60+ columns")))),
	)
}

// wire installs the view and key handling. FilterList registers its own text input and
// <C-n>/<C-p>/<C-d>/<C-u> on the shared view router (last list wins), so every key is
// re-registered here after SetView, focus-aware, which overrides them.
func wire(app *App, s *state) {
	app.SetView(rootView(s))
	app.OnResize(func(w, h int) { s.resize(w, h) })

	r := app.Router()
	r.NoCounts()
	r.HandleUnmatched(s.handleText)

	app.Handle("<Up>", func() { s.move(-1) })
	app.Handle("<C-p>", func() { s.move(-1) })
	app.Handle("<Down>", func() { s.move(1) })
	app.Handle("<C-n>", func() { s.move(1) })
	app.Handle("<PageUp>", func() { s.page(-1) })
	app.Handle("<PageDown>", func() { s.page(1) })
	app.Handle("<Enter>", s.enter)
	app.Handle("<Right>", s.enter)
	app.Handle("<Esc>", s.back)
	app.Handle("<Left>", s.back)
	app.Handle("<Tab>", func() { s.tick(1) })
	app.Handle("<S-Tab>", func() { s.tick(-1) })
	app.Handle("<C-a>", s.tickAllMatching)
	app.Handle("<C-c>", app.Stop)
}

// Run builds the app and blocks until the user quits.
func Run(d Deps) error {
	app := NewApp()
	s := newState(d, func(f func()) { go f() }, app.Apply, app.RequestRender)
	wire(app, s)
	return app.Run()
}
