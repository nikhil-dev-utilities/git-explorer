package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kungfusheep/riffkey"

	"github.com/nikhil-dev-utilities/git-explorer/internal/forge"
)

func org(name string, a forge.Affiliation) forge.Org { return forge.Org{Name: name, Affiliation: a} }

func repo(org, name string) forge.Repo { return forge.Repo{Org: org, Name: name} }

func defaultFake() *fakeForge {
	return &fakeForge{
		pages: map[string][]forge.OrgPage{
			"github.com": {
				{Orgs: []forge.Org{org("globex", forge.AffiliationMember), org("acme", forge.AffiliationOwner)}},
				{Orgs: []forge.Org{org("platform-eng", forge.AffiliationCollaborator)}},
			},
			"ghe.corp": {{Orgs: []forge.Org{org("corp", forge.AffiliationNone)}}},
		},
		repos: map[string][]forge.Repo{
			"acme": {
				repo("acme", "tf-network"),
				repo("acme", "tf-dns"),
				{Org: "acme", Name: "old-thing", Archived: true},
				{Org: "acme", Name: "a-fork", Fork: true},
				repo("acme", "web"),
			},
		},
	}
}

func newTest(t *testing.T, f *fakeForge, userConfigured bool) *state {
	t.Helper()
	d := Deps{
		Forge:               f,
		Hosts:               []forge.Host{{Name: "github.com"}, {Name: "ghe.corp"}},
		HostsUserConfigured: userConfigured,
		Parallelism:         4,
	}
	sync := func(fn func()) { fn() }
	s := newState(d, sync, sync, func() {})
	return s
}

func typeText(s *state, text string) {
	for _, r := range text {
		s.handleText(riffkey.Key{Rune: r})
	}
}

func orgNames(s *state) []string {
	var out []string
	for _, r := range s.orgFL.Filter().Items {
		out = append(out, r.Org.Name)
	}
	return out
}

func repoNames(s *state) []string {
	var out []string
	for _, r := range s.repoFL.Filter().Items {
		out = append(out, r.Repo.Name)
	}
	return out
}

func join(x []string) string { return strings.Join(x, ",") }

func TestOrgsLoadAcrossPagesSortedByName(t *testing.T) {
	s := newTest(t, defaultFake(), true)
	if got := join(orgNames(s)); got != "acme,globex,platform-eng" {
		t.Errorf("orgs = %s", got)
	}
	if !s.orgsLoaded {
		t.Error("orgsLoaded not set after the stream closed")
	}
}

func TestFilterIsFzfSyntaxAndAlwaysLive(t *testing.T) {
	s := newTest(t, defaultFake(), true)

	typeText(s, "gx") // fuzzy subsequence
	if got := join(orgNames(s)); got != "globex" {
		t.Errorf("fuzzy gx = %s", got)
	}
	s.clearQuery(focusOrgs)

	typeText(s, "^pl") // prefix
	if got := join(orgNames(s)); got != "platform-eng" {
		t.Errorf("^pl = %s", got)
	}
	s.clearQuery(focusOrgs)

	typeText(s, "'ob") // exact substring
	if got := join(orgNames(s)); got != "globex" {
		t.Errorf("'ob = %s", got)
	}
	s.clearQuery(focusOrgs)

	typeText(s, "!e") // negated
	if got := join(orgNames(s)); got != "" {
		t.Errorf("!e = %q, every org name contains e", got)
	}
	if !strings.Contains(s.orgMsg, "no matches") {
		t.Errorf("orgMsg = %q", s.orgMsg)
	}
}

func TestDescendLoadsReposHidesArchivedAndForksByDefault(t *testing.T) {
	f := defaultFake()
	s := newTest(t, f, true)
	s.enter() // first org: acme
	if s.focus != focusRepos || s.currentOrg.Name != "acme" {
		t.Fatalf("focus=%v org=%q", s.focus, s.currentOrg.Name)
	}
	if got := join(repoNames(s)); got != "tf-dns,tf-network,web" {
		t.Errorf("repos = %s, want archived and fork hidden", got)
	}
	if strings.Join(f.repoCalls, ",") != "acme" {
		t.Errorf("repoCalls = %v", f.repoCalls)
	}

	s.archived = triShow
	s.fork = triShow
	s.rebuildRepos()
	if len(repoNames(s)) != 5 {
		t.Errorf("repos with facets shown = %v", repoNames(s))
	}
	s.archived = triOnly
	s.fork = triHide
	s.rebuildRepos()
	if got := join(repoNames(s)); got != "old-thing" {
		t.Errorf("archived only = %s", got)
	}
}

func TestEscInReposGoesBackAndKeepsOrgFilter(t *testing.T) {
	s := newTest(t, defaultFake(), true)
	typeText(s, "ac")
	s.enter()
	s.back()
	if s.focus != focusOrgs || s.query[focusOrgs] != "ac" {
		t.Errorf("focus=%v query=%q, want orgs focused, filter intact", s.focus, s.query[focusOrgs])
	}
	s.back() // Esc in Orgs clears the filter
	if s.query[focusOrgs] != "" || len(orgNames(s)) != 3 {
		t.Errorf("Org filter not cleared: %q %v", s.query[focusOrgs], orgNames(s))
	}
}

func TestTickAllMatchesOnlyTheFilteredRepos(t *testing.T) {
	s := newTest(t, defaultFake(), true)
	s.enter()
	typeText(s, "tf")
	s.tickAllMatching()
	if s.selectionCount() != 2 || s.selected["web"] {
		t.Errorf("selected = %v, want only tf-*", s.selected)
	}
}

func TestLeavePromptGuardsSelection(t *testing.T) {
	s := newTest(t, defaultFake(), true)
	s.enter()
	s.tick(1)
	s.back()
	if s.mode != modeLeave || s.focus != focusRepos {
		t.Fatalf("mode=%v focus=%v, want leave prompt with focus kept", s.mode, s.focus)
	}

	s.leaveStay()
	if s.mode != modeBrowse || s.selectionCount() != 1 {
		t.Errorf("stay: mode=%v sel=%d", s.mode, s.selectionCount())
	}

	s.back()
	s.leaveDiscard()
	if s.mode != modeBrowse || s.focus != focusOrgs || s.selectionCount() != 0 {
		t.Errorf("discard: mode=%v focus=%v sel=%d", s.mode, s.focus, s.selectionCount())
	}
}

func TestEnterInReposWithNothingTickedDoesNothing(t *testing.T) {
	s := newTest(t, defaultFake(), true)
	s.enter()
	s.enter()
	if s.mode != modeBrowse {
		t.Errorf("mode = %v", s.mode)
	}
	s.tick(1)
	s.enter()
	if s.mode != modeClone {
		t.Errorf("mode = %v, want clone", s.mode)
	}
}

func TestHostSwitchResetsEverythingAndReloads(t *testing.T) {
	f := defaultFake()
	s := newTest(t, f, true)
	typeText(s, "ac")
	s.enter()
	s.tick(1)

	s.openHostSwitch()
	if s.mode != modeHost || s.hostCursor != 0 {
		t.Fatalf("mode=%v cursor=%d", s.mode, s.hostCursor)
	}
	s.moveHost(1)
	s.moveHost(5) // clamps
	s.confirmHost()

	if s.mode != modeBrowse || s.activeHost().Name != "ghe.corp" || s.focus != focusOrgs {
		t.Fatalf("mode=%v host=%s focus=%v", s.mode, s.activeHost().Name, s.focus)
	}
	if got := join(orgNames(s)); got != "corp" {
		t.Errorf("orgs = %s, want only the new host's", got)
	}
	if s.selectionCount() != 0 || len(repoNames(s)) != 0 || s.query[focusOrgs] != "" || s.query[focusRepos] != "" {
		t.Errorf("state leaked across hosts: sel=%d repos=%v q=%v", s.selectionCount(), repoNames(s), s.query)
	}
	if strings.Join(f.orgCalls, ",") != "github.com,ghe.corp" {
		t.Errorf("orgCalls = %v", f.orgCalls)
	}

	s.openHostSwitch()
	s.moveHost(-1)
	s.cancelHost()
	if s.activeHost().Name != "ghe.corp" {
		t.Errorf("cancel changed the host")
	}
}

func TestFailureSurfaces(t *testing.T) {
	fatal := &forge.Error{Kind: forge.ErrKindFatal, Message: "not authenticated: run gh auth login"}

	t.Run("fatal when the user configured the host", func(t *testing.T) {
		s := newTest(t, &fakeForge{orgsErr: fatal}, true)
		if s.mode != modeFatal || !strings.Contains(s.fatalText, "gh auth login") || !s.showFatal {
			t.Errorf("mode=%v text=%q", s.mode, s.fatalText)
		}
	})

	t.Run("fatal downgrades to pane error for a discovered host", func(t *testing.T) {
		s := newTest(t, &fakeForge{orgsErr: fatal}, false)
		if s.mode != modeBrowse || s.orgsErr == nil || !strings.Contains(s.orgMsg, "error loading orgs") {
			t.Errorf("mode=%v orgsErr=%v msg=%q", s.mode, s.orgsErr, s.orgMsg)
		}
	})

	t.Run("pane error mid-stream keeps loaded orgs", func(t *testing.T) {
		f := &fakeForge{pages: map[string][]forge.OrgPage{"github.com": {
			{Orgs: []forge.Org{org("acme", forge.AffiliationOwner)}},
			{Err: errors.New("network blip")},
		}}}
		s := newTest(t, f, true)
		if join(orgNames(s)) != "acme" || s.orgsErr == nil || !strings.Contains(s.orgMsg, "showing what loaded") {
			t.Errorf("orgs=%v err=%v msg=%q", orgNames(s), s.orgsErr, s.orgMsg)
		}
	})

	t.Run("transient goes to the status line", func(t *testing.T) {
		rl := &forge.Error{Kind: forge.ErrKindTransient, Message: "rate limited", RetryAfter: 30 * time.Second}
		s := newTest(t, &fakeForge{orgsErr: rl}, true)
		if !strings.Contains(s.status, "rate limited") || !strings.Contains(s.status, "30s") || s.orgsErr != nil {
			t.Errorf("status=%q orgsErr=%v", s.status, s.orgsErr)
		}
	})

	t.Run("repo load error is pane scoped and reload retries", func(t *testing.T) {
		f := defaultFake()
		f.reposErr = errors.New("boom")
		s := newTest(t, f, true)
		s.enter()
		if s.reposErr == nil || !strings.Contains(s.repoMsg, "error loading repos") {
			t.Fatalf("reposErr=%v msg=%q", s.reposErr, s.repoMsg)
		}
		f.reposErr = nil
		s.reload()
		if s.reposErr != nil || len(repoNames(s)) != 3 {
			t.Errorf("reload: err=%v repos=%v", s.reposErr, repoNames(s))
		}
	})
}

func TestStaleLoadsAreDropped(t *testing.T) {
	var queue []func()
	d := Deps{Forge: defaultFake(), Hosts: []forge.Host{{Name: "github.com"}}, HostsUserConfigured: true}
	s := newState(d, func(f func()) { queue = append(queue, f) }, func(f func()) { f() }, func() {})
	first := queue[0]
	s.loadOrgs() // a second load supersedes the first
	second := queue[1]
	second()
	first() // late arrival from the superseded load must not append
	if got := len(s.orgAll); got != 3 {
		t.Errorf("orgAll = %d, want 3 (stale load appended)", got)
	}
}

func TestPaneMessagesAreDistinct(t *testing.T) {
	cases := map[string]paneInfo{
		"loading...":                     {what: "orgs"},
		"no orgs":                        {what: "orgs", loaded: true},
		`no matches for "x"`:             {what: "orgs", loaded: true, all: 3, query: "x"},
		"nothing matches the current op": {what: "repos", loaded: true, all: 3},
	}
	seen := map[string]bool{}
	for want, p := range cases {
		got := paneMessage(p)
		if !strings.HasPrefix(got, want) {
			t.Errorf("paneMessage(%+v) = %q, want prefix %q", p, got, want)
		}
		if seen[got] {
			t.Errorf("duplicate message %q", got)
		}
		seen[got] = true
	}
	if paneMessage(paneInfo{what: "orgs", loaded: true, all: 3, visible: 2}) != "" {
		t.Error("message shown although rows are visible")
	}
}

func TestLayoutSheddingAndTooNarrow(t *testing.T) {
	if !layoutFor(59, 1).tooNarrow || layoutFor(60, 1).tooNarrow {
		t.Error("tooNarrow threshold is 60 columns")
	}
	wide := layoutFor(140, 1)
	if !wide.showAge || !wide.showBadges || wide.orgWidth != 34 {
		t.Errorf("wide = %+v", wide)
	}
	mid := layoutFor(90, 1)
	if mid.showAge || !mid.showBadges {
		t.Errorf("mid = %+v, want badges without age", mid)
	}
	narrow := layoutFor(62, 3) // widest preset squeezed
	if narrow.showBadges || narrow.orgWidth != int16(62-4-minRepoPaneWidth) {
		t.Errorf("narrow = %+v, want name only and a shrunk org pane", narrow)
	}
}

func TestAgo(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "just now"}, {5 * time.Minute, "5m ago"}, {3 * time.Hour, "3h ago"},
		{2 * 24 * time.Hour, "2d ago"}, {90 * 24 * time.Hour, "3mo ago"}, {800 * 24 * time.Hour, "2y ago"},
	} {
		if got := ago(now, now.Add(-tc.d)); got != tc.want {
			t.Errorf("ago(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
	if ago(now, time.Time{}) != "" {
		t.Error("zero time should render empty")
	}
}

func testNow() time.Time { return time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC) }
