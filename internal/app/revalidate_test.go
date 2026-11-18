package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

func onCommentsTab(t *testing.T) App {
	t.Helper()
	a := newTestApp(t, 160, 45)
	a.rightTab = PanelComments
	return a
}

func selectedWorkItem(t *testing.T, a App) int {
	t.Helper()
	item := a.selectedItem()
	if item == nil {
		t.Fatal("no work item selected")
	}
	return item.ID
}

func reselect(a *App) tea.Cmd {
	a.commentsFor = 0
	return a.syncSelection()
}

func TestRevisitedCommentsShowAtOnceAndOnlyRefreshWhenOld(t *testing.T) {
	a := onCommentsTab(t)
	clock := &testClock{now: time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)}
	a.comments.SetClock(clock.Now)
	id := selectedWorkItem(t, a)
	a.comments.Put(id, []models.Comment{{ID: 1, Text: "Ship it behind a flag", CreatedBy: "Lena Ortiz"}})

	if cmd := a.syncSelection(); cmd != nil {
		t.Fatal("selection did not change, nothing to load")
	}
	if cmd := reselect(&a); cmd != nil {
		t.Fatal("comments read seconds ago must not be fetched again")
	}
	if a.commentsPanel.Loading() || !strings.Contains(ansi.Strip(a.render()), "Ship it behind a flag") {
		t.Fatal("cached comments must show immediately, without a spinner")
	}

	clock.now = clock.now.Add(revalidateAfter + time.Second)
	if cmd := reselect(&a); cmd == nil {
		t.Fatal("comments older than the revalidation window must be refreshed in the background")
	}
	if a.commentsPanel.Loading() {
		t.Fatal("while refreshing, the cached comments stay on screen")
	}
}

func TestAFailedRefreshKeepsTheCachedComments(t *testing.T) {
	a := onCommentsTab(t)
	id := selectedWorkItem(t, a)
	a.comments.Put(id, []models.Comment{{ID: 1, Text: "Ship it behind a flag", CreatedBy: "Lena Ortiz"}})
	a, _ = step(t, a, commentsLoadedMsg{workItemID: id, err: errors.New("API error 503")})
	if got, _ := a.comments.Get(id); len(got) != 1 {
		t.Fatal("a failed refresh must not replace good comments")
	}
	if !strings.Contains(ansi.Strip(a.render()), "Ship it behind a flag") {
		t.Fatal("a failed refresh keeps showing the cached comments")
	}
}

func TestChangingACommentForgetsTheCache(t *testing.T) {
	a := newTestApp(t, 160, 45)
	id := selectedWorkItem(t, a)
	a.comments.Put(id, []models.Comment{{ID: 1, Text: "old"}})
	a, _ = step(t, a, commentChangedMsg{workItemID: id, notice: "Comment added"})
	if _, ok := a.comments.Get(id); ok {
		t.Fatal("after your own change the cached comments must be dropped so they are never shown stale")
	}
}

func settledChecks() []models.PullRequestCheck {
	return []models.PullRequestCheck{{Name: "Build: storefront-ci", State: models.CheckPassed, Required: true}}
}

func reselectPullRequest(a *App) tea.Cmd {
	a.pr.checksFor = 0
	return a.syncPullRequestSelection()
}

func TestSettledChecksShowFromCacheWithoutARequest(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	id := a.pr.checksFor
	a, _ = step(t, a, checksLoadedMsg{id: id, checks: settledChecks()})
	if cmd := reselectPullRequest(&a); cmd != nil {
		t.Fatal("finished checks read seconds ago must not be fetched again")
	}
	if !a.pr.checksLoaded || len(a.pr.checks) != 1 {
		t.Fatal("cached checks must show immediately")
	}
}

func TestRunningChecksAreAlwaysRefreshed(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	if cmd := reselectPullRequest(&a); cmd == nil {
		t.Fatal("a pending or running check must never be served from the cache alone")
	}
	if !a.pr.checksLoaded || len(a.pr.checks) != len(sampleChecks()) {
		t.Fatal("the last known checks still show while they refresh")
	}
}

func TestOldSettledChecksAreRefreshed(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	clock := &testClock{now: time.Now()}
	a.pr.checksCache.SetClock(clock.Now)
	a, _ = step(t, a, checksLoadedMsg{id: a.pr.checksFor, checks: settledChecks()})
	clock.now = clock.now.Add(revalidateAfter + time.Second)
	if cmd := reselectPullRequest(&a); cmd == nil {
		t.Fatal("finished checks older than the window must be refreshed")
	}
}

func TestAFailedChecksLoadKeepsTheCachedChecks(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	id := a.pr.checksFor
	a, _ = step(t, a, checksLoadedMsg{id: id, err: errors.New("API error 500")})
	if len(a.pr.checks) != len(sampleChecks()) {
		t.Fatalf("got %d checks after a failed refresh, want the %d cached ones", len(a.pr.checks), len(sampleChecks()))
	}
}

func TestSwitchingBackToAPivotShowsTheCachedList(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	a, _ = step(t, a, press("3"))
	if !a.pr.loading {
		t.Fatal("Completed was never loaded and must load")
	}
	completed := []models.PullRequest{samplePullRequests()[1]}
	completed[0].Status = models.PullRequestCompleted
	a, _ = step(t, a, pullRequestsLoadedMsg{seq: a.pr.seq, status: models.PullRequestCompleted, items: completed})

	seq := a.pr.seq
	a, _ = step(t, a, press("2"))
	if a.pr.loading || a.pr.panel.Selected() == nil {
		t.Fatal("Active was loaded seconds ago and must show at once without loading")
	}
	if !strings.Contains(ansi.Strip(a.render()), "Fix null tenant id") {
		t.Fatal("the cached Active list must be on screen")
	}
	if a.pr.seq == seq {
		t.Fatal("showing a cached list must invalidate any load still in flight for another pivot")
	}
}

func TestAnOldCachedListRefreshesInTheBackground(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	clock := &testClock{now: time.Now()}
	a.pr.lists.SetClock(clock.Now)
	a.pr.lists.Put(a.pr.listKey(models.PullRequestActive), samplePullRequests())
	a, _ = step(t, a, press("3"))
	clock.now = clock.now.Add(revalidateAfter + time.Second)
	a, _ = step(t, a, press("2"))
	if !a.pr.loading || !a.pr.background || a.pr.panel.Selected() == nil {
		t.Fatal("an old cached list must stay on screen while it refreshes in the background")
	}
	a, _ = step(t, a, pullRequestsFailedMsg{seq: a.pr.seq, err: errors.New("API error 503")})
	if a.modals.open() || a.pr.panel.Selected() == nil {
		t.Fatal("a failed background refresh keeps the cached list and does not interrupt")
	}
}

func TestAFailedExplicitRefreshStillReports(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	clock := &testClock{now: time.Now()}
	a.pr.lists.SetClock(clock.Now)
	a, _ = step(t, a, press("3"))
	clock.now = clock.now.Add(revalidateAfter + time.Second)
	a, _ = step(t, a, press("2"))
	if !a.pr.background {
		t.Fatal("setup: Active should be refreshing in the background")
	}
	seq := a.pr.seq
	a, _ = step(t, a, ctrl('r'))
	if a.pr.seq == seq || a.pr.background {
		t.Fatal("Ctrl+r must start a fresh, foreground load")
	}
	a, _ = step(t, a, pullRequestsFailedMsg{seq: a.pr.seq, err: errors.New("API error 503")})
	if !a.modals.open() {
		t.Fatal("a refresh you asked for must report its failure")
	}
}
