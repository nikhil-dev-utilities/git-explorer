package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

// orgsStreamMsg carries the channel ListOrgs returned, once the synchronous part of
// the call (which may fail fatally — not authenticated, gh missing) has succeeded.
type orgsStreamMsg struct {
	ch <-chan forge.OrgPage
}

// orgsFatalErrMsg carries ListOrgs's synchronous error. A later slice of this PRD
// (failure surfaces) renders this as the full-screen Fatal mode; for now it is
// recorded on the Model without a dedicated rendering.
type orgsFatalErrMsg struct {
	err error
}

// orgPageMsg is one value read off the OrgPage stream: either a page (possibly
// carrying its own pane-scoped error, per forge.OrgPage's contract) or, when done is
// true, notice that the stream has closed.
type orgPageMsg struct {
	page forge.OrgPage
	ch   <-chan forge.OrgPage
	done bool
}

// listOrgsCmd calls the injected Forge's ListOrgs. Per Forge's contract, a
// synchronous error here means nothing was fetched at all — typically fatal (not
// authenticated, gh missing) — while pages arriving over the channel might separately
// end in an OrgPage carrying its own Err (pane-scoped: some pages loaded, one failed).
func listOrgsCmd(f forge.Forge, host forge.Host) tea.Cmd {
	return func() tea.Msg {
		ch, err := f.ListOrgs(backgroundCtx(), host)
		if err != nil {
			return orgsFatalErrMsg{err: err}
		}
		return orgsStreamMsg{ch: ch}
	}
}

// readNextPageCmd reads exactly one value off ch, so the Model can render each page
// as it arrives rather than waiting for the whole stream — the point of ListOrgs
// being a stream at all (DESIGN.md, "No cache" / progressive loading).
func readNextPageCmd(ch <-chan forge.OrgPage) tea.Cmd {
	return func() tea.Msg {
		page, ok := <-ch
		if !ok {
			return orgPageMsg{done: true}
		}
		return orgPageMsg{page: page, ch: ch}
	}
}

func (m Model) handleOrgsStream(msg orgsStreamMsg) (Model, tea.Cmd) {
	m.orgsCh = msg.ch
	return m, readNextPageCmd(msg.ch)
}

func (m Model) handleOrgPage(msg orgPageMsg) (Model, tea.Cmd) {
	if msg.done {
		m.orgsLoaded = true
		return m, nil
	}
	if msg.page.Err != nil {
		// Whatever loaded on earlier pages (already appended to m.orgs on prior
		// calls) is deliberately left untouched — applyOrgsFailure only sets
		// orgsLoaded and the classified error field.
		return m.applyOrgsFailure(msg.page.Err), nil
	}
	m.orgs = append(m.orgs, msg.page.Orgs...)
	return m, readNextPageCmd(msg.ch)
}

func (m Model) handleOrgsFatalErr(msg orgsFatalErrMsg) (Model, tea.Cmd) {
	return m.applyOrgsFailure(msg.err), nil
}
