package ui

import (
	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"

	"github.com/nikhil-dev-utilities/git-explorer/internal/glyphclone"
)

var modalFill = RGB(0x1c, 0x1c, 0x1c)

func orgPane(s *state) Component {
	s.orgFL.Placeholder("press / to filter").Render(func(r *orgRow) Component {
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
	s.repoFL.Placeholder("press / to filter").Render(func(r *repoRow) Component {
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

func leaveCard(s *state) Component {
	return VBox.Border(BorderRounded).Fill(modalFill).Padding(1).FitContent()(
		Text(&s.leaveText).Bold(),
		Text(""),
		Text("c clone now · d discard and continue · esc stay").Dim(),
	)
}

func hostCard(s *state) Component {
	return VBox.Border(BorderRounded).Title("Hosts").Fill(modalFill).Padding(1).FitContent()(
		List(&s.hostRows).Selection(&s.hostCursor).Render(func(r *hostRow) Component {
			return HBox(Text(&r.Mark), SpaceW(1), Text(&r.Name))
		}),
		Text(""),
		Text("↑↓ move · enter switch · esc cancel · * active").Dim(),
	)
}

func helpCard(s *state) Component {
	return VBox.Border(BorderRounded).Title("Help").Fill(modalFill).Padding(1).Height(&s.helpH)(
		List(&s.helpRows).Selection(&s.helpCursor).Marker("").Render(func(r *helpRow) Component {
			return HBox(Text(&r.Key).Width(16), Text(&r.Text))
		}),
		Text("↑↓ scroll · esc close").Dim(),
	)
}

func fatalView(s *state) Component {
	return VBox.Grow(1)(
		Text("git-explorer cannot continue").Bold().FG(Red),
		Text(""),
		Text(&s.fatalText),
		Text(""),
		Text("y switch host · ^c quit").Dim(),
	)
}

// rootView keeps the panes outside any conditional: Glyph wires a conditional
// branch's key bindings through child scopes, which would sit ahead of the routing
// override in wire. Full-screen states are overlays drawn over the panes instead.
func rootView(s *state, clone *glyphclone.Embedded) Component {
	return VBox.Grow(1)(
		browseView(s),
		If(&s.showOptions).Then(Overlay.Backdrop().Centered()(optionsCard(s))),
		If(&s.showLeave).Then(Overlay.Backdrop().Centered()(leaveCard(s))),
		If(&s.showHost).Then(Overlay.Backdrop().Centered()(hostCard(s))),
		If(&s.showHelp).Then(Overlay.Backdrop().Centered()(helpCard(s))),
		If(&s.showClone).Then(Overlay.At(0, 0)(VBox.Width(&s.screenW).Height(&s.screenH).Fill(modalFill)(clone.View()))),
		If(&s.showFatal).Then(Overlay.Backdrop().Centered()(VBox.Border(BorderRounded).Fill(modalFill).Padding(1).FitContent()(fatalView(s)))),
		If(&s.showTooNarrow).Then(Overlay.Backdrop().Centered()(VBox.Border(BorderRounded).Fill(modalFill).Padding(1).FitContent()(Text("terminal too narrow: widen to 60+ columns")))),
	)
}

// wire installs the view and key handling. FilterList registers its own text input and
// <C-n>/<C-p>/<C-d>/<C-u> on the shared view router (last list wins), so every key is
// re-registered here after SetView, focus-aware, which overrides them. Each modal mode
// (including typing a filter) owns a router pushed on entry and popped on exit, so browse
// keys are inert under it and bare letters are text only while filtering (ADR-0009).
func wire(app *App, s *state) {
	scr := glyphclone.Embed(glyphclone.Hooks{
		Spawn: s.spawn, Apply: s.apply, Refresh: s.refresh,
		Push: app.PushRouter, Pop: app.PopRouter, Done: s.cloneDone,
	})
	s.openScreen = scr.Open
	app.SetView(rootView(s, scr))
	app.OnResize(func(w, h int) { s.resize(w, h) })
	app.OnBeforeRender(s.updateTitles)

	bind := func(r *riffkey.Router, pattern string, fn func()) {
		r.Handle(pattern, func(riffkey.Match) { fn(); app.RequestRender() })
	}
	bindAll := func(r *riffkey.Router, fn func(), patterns ...string) {
		for _, p := range patterns {
			bind(r, p, fn)
		}
	}
	quit := func(r *riffkey.Router) { r.Handle("<C-c>", func(riffkey.Match) { app.Stop() }) }
	moveKeys := func(r *riffkey.Router) {
		bindAll(r, func() { s.move(-1) }, "<Up>", "<C-p>")
		bindAll(r, func() { s.move(1) }, "<Down>", "<C-n>")
		bind(r, "<PageUp>", func() { s.page(-1) })
		bind(r, "<PageDown>", func() { s.page(1) })
	}

	base := app.Router()
	base.NoCounts()
	// Swallow keys nothing binds: FilterList's own text binding is still registered here
	// and would otherwise type bare letters into the last pane's hidden input.
	base.HandleUnmatched(func(riffkey.Key) bool { return false })
	moveKeys(base)
	bind(base, "j", func() { s.move(1) })
	bind(base, "k", func() { s.move(-1) })
	bind(base, "/", s.startFilter)
	bindAll(base, s.cyclePane, "<Tab>", "<S-Tab>")
	bind(base, "<Enter>", s.enter)
	bindAll(base, s.right, "<Right>", "l")
	bindAll(base, s.left, "<Left>", "h")
	bind(base, "<Esc>", s.back)
	bind(base, "<Space>", func() { s.tick(1) })
	bind(base, "a", s.toggleAllMatching)
	bind(base, "x", s.clearSelection)
	bind(base, "c", s.cloneSelection)
	bindAll(base, s.reload, "r", "<F5>")
	bindAll(base, s.openOptions, "o", "<C-o>")
	bindAll(base, s.openHelp, "?", "<F1>")
	quit(base)

	filter := riffkey.NewRouter().NoCounts()
	filter.HandleUnmatched(s.handleText)
	moveKeys(filter)
	bind(filter, "<Enter>", s.acceptFilter)
	bind(filter, "<Esc>", s.cancelFilter)
	quit(filter)

	options := riffkey.NewRouter().NoCounts()
	moveMenu := func(delta int) func() { return func() { s.menuMove(delta) } }
	bindAll(options, moveMenu(-1), "<Up>", "<C-p>", "k")
	bindAll(options, moveMenu(1), "<Down>", "<C-n>", "j")
	bindAll(options, s.menuActivate, "<Enter>", "<Space>", "<Right>")
	bindAll(options, s.closeOptions, "<Esc>", "<C-o>", "o")
	quit(options)

	leave := riffkey.NewRouter().NoCounts()
	bind(leave, "c", s.leaveClone)
	bind(leave, "d", s.leaveDiscard)
	bind(leave, "<Esc>", s.leaveStay)
	quit(leave)

	hosts := riffkey.NewRouter().NoCounts()
	bindAll(hosts, func() { s.moveHost(-1) }, "<Up>", "<C-p>", "k")
	bindAll(hosts, func() { s.moveHost(1) }, "<Down>", "<C-n>", "j")
	bind(hosts, "<Enter>", s.confirmHost)
	bind(hosts, "<Esc>", s.cancelHost)
	quit(hosts)

	help := riffkey.NewRouter().NoCounts()
	bindAll(help, func() { s.helpMove(-1) }, "<Up>", "k")
	bindAll(help, func() { s.helpMove(1) }, "<Down>", "j")
	bind(help, "<PageUp>", func() { s.helpMove(-10) })
	bind(help, "<PageDown>", func() { s.helpMove(10) })
	bindAll(help, s.closeHelp, "<Esc>", "<F1>", "?")
	quit(help)

	fatal := riffkey.NewRouter().NoCounts()
	bind(fatal, "<C-y>", s.openHostSwitch)
	bind(fatal, "y", s.openHostSwitch)
	quit(fatal)

	modal := map[mode]*riffkey.Router{
		modeOptions: options, modeLeave: leave, modeHost: hosts, modeHelp: help, modeFatal: fatal,
		modeClone: scr.Router(), modeFilter: filter,
	}
	s.onMode = func(prev, next mode) {
		if prev != modeBrowse {
			app.PopRouter()
		}
		if r := modal[next]; next != modeBrowse && r != nil {
			app.PushRouter(r)
		}
	}
	// a load that failed fatally before wire ran (synchronous seams in tests)
	if s.mode != modeBrowse {
		s.onMode(modeBrowse, s.mode)
	}
}

// Run builds the app and blocks until the user quits.
func Run(d Deps) error {
	app := NewApp()
	s := newState(d, func(f func()) { go f() }, app.Apply, app.RequestRender)
	wire(app, s)
	return app.Run()
}
