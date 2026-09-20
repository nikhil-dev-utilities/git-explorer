package ui

import (
	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
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

func optionsCard(s *state) Component {
	return VBox.Border(BorderRounded).Title("Options").Fill(modalFill).Padding(1).FitContent()(
		List(&s.menu).Selection(&s.menuCursor).Render(func(r *menuRow) Component {
			return HBox(Text(&r.Label), SpaceW(2), Space(), Text(&r.Value).FG(Cyan))
		}),
		Text(""),
		Text("↑↓ move · enter change · esc close").Dim(),
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
		If(&s.showOptions).Then(Overlay.Backdrop().Centered()(optionsCard(s))),
		If(&s.showFatal).Then(Overlay.Backdrop().Centered()(VBox.Border(BorderRounded).Fill(modalFill).Padding(1).FitContent()(fatalView(s)))),
		If(&s.showTooNarrow).Then(Overlay.Backdrop().Centered()(VBox.Border(BorderRounded).Fill(modalFill).Padding(1).FitContent()(Text("terminal too narrow: widen to 60+ columns")))),
	)
}

// wire installs the view and key handling. FilterList registers its own text input and
// <C-n>/<C-p>/<C-d>/<C-u> on the shared view router (last list wins), so every key is
// re-registered here after SetView, focus-aware, which overrides them. Each modal mode
// owns a router pushed on entry and popped on exit, so browse keys are inert under it.
func wire(app *App, s *state) {
	app.SetView(rootView(s))
	app.OnResize(func(w, h int) { s.resize(w, h) })

	bind := func(r *riffkey.Router, pattern string, fn func()) {
		r.Handle(pattern, func(riffkey.Match) { fn(); app.RequestRender() })
	}

	base := app.Router()
	base.NoCounts()
	base.HandleUnmatched(s.handleText)
	bind(base, "<Up>", func() { s.move(-1) })
	bind(base, "<C-p>", func() { s.move(-1) })
	bind(base, "<Down>", func() { s.move(1) })
	bind(base, "<C-n>", func() { s.move(1) })
	bind(base, "<PageUp>", func() { s.page(-1) })
	bind(base, "<PageDown>", func() { s.page(1) })
	bind(base, "<Enter>", s.enter)
	bind(base, "<Right>", s.enter)
	bind(base, "<Esc>", s.back)
	bind(base, "<Left>", s.back)
	bind(base, "<Tab>", func() { s.tick(1) })
	bind(base, "<S-Tab>", func() { s.tick(-1) })
	bind(base, "<C-a>", s.tickAllMatching)
	bind(base, "<C-o>", s.openOptions)
	base.Handle("<C-c>", func(riffkey.Match) { app.Stop() })

	options := riffkey.NewRouter().NoCounts()
	bind(options, "<Up>", func() { s.menuMove(-1) })
	bind(options, "<Down>", func() { s.menuMove(1) })
	bind(options, "<C-p>", func() { s.menuMove(-1) })
	bind(options, "<C-n>", func() { s.menuMove(1) })
	bind(options, "<Enter>", s.menuActivate)
	bind(options, "<Space>", s.menuActivate)
	bind(options, "<Right>", s.menuActivate)
	bind(options, "<Esc>", s.closeOptions)
	bind(options, "<C-o>", s.closeOptions)
	options.Handle("<C-c>", func(riffkey.Match) { app.Stop() })

	modal := map[mode]*riffkey.Router{modeOptions: options}
	s.onMode = func(prev, next mode) {
		if prev != modeBrowse {
			app.PopRouter()
		}
		if r := modal[next]; next != modeBrowse && r != nil {
			app.PushRouter(r)
		}
	}
}

// Run builds the app and blocks until the user quits.
func Run(d Deps) error {
	app := NewApp()
	s := newState(d, func(f func()) { go f() }, app.Apply, app.RequestRender)
	wire(app, s)
	return app.Run()
}
