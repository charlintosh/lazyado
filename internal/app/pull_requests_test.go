package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

var reviewer = models.Identity{ID: "me-id", DisplayName: "Dana Reyes", UniqueName: "dana@example.com"}

func samplePullRequests() []models.PullRequest {
	now := time.Now()
	pr := func(id int, title, author, authorID, source string, draft bool, reviewers ...models.Reviewer) models.PullRequest {
		return models.PullRequest{
			ID:             id,
			Title:          title,
			Description:    "## Why\n\nWebhook deliveries fail on transient errors and are never retried.\n\n- Exponential backoff\n- Delivery log",
			Status:         models.PullRequestActive,
			IsDraft:        draft,
			Author:         models.Identity{ID: authorID, DisplayName: author},
			SourceBranch:   source,
			TargetBranch:   "main",
			RepositoryID:   "repo-1",
			RepositoryName: "storefront-api",
			ProjectID:      "project-1",
			CreatedDate:    now.Add(-3 * time.Hour),
			Reviewers:      reviewers,
			WebURL:         "https://dev.azure.com/acme/storefront/_git/storefront-api/pullrequest/1",
		}
	}
	me := models.Reviewer{ID: "me-id", DisplayName: "Dana Reyes", Required: true}
	approved := models.Reviewer{ID: "u2", DisplayName: "Marco Lima", Vote: models.VoteApproved, Required: true}
	waiting := models.Reviewer{ID: "u3", DisplayName: "Priya Nair", Vote: models.VoteWaitingForAuthor}
	return []models.PullRequest{
		pr(4821, "Add retry policy to payment webhook client", "Sam Kowal", "u4", "feature/webhook-retry", false, me, approved, waiting),
		pr(4817, "Fix null tenant id in audit export", "Marco Lima", "u2", "bugfix/audit-tenant", false, approved),
		pr(4809, "Migrate invoice templates to Liquid", "Dana Reyes", "me-id", "feature/liquid-invoices", true, waiting),
	}
}

func sampleChecks() []models.PullRequestCheck {
	return []models.PullRequestCheck{
		{Name: "Build: storefront-ci", State: models.CheckPassed, Required: true},
		{Name: "Minimum number of reviewers", Detail: "2 required", State: models.CheckPending, Required: true},
		{Name: "sonarqube/quality-gate", Detail: "Coverage on new code below gate", State: models.CheckFailed},
	}
}

func openPullRequestHub(t testing.TB, width, height int) App {
	t.Helper()
	a := newTestAppAs(t, width, height, reviewer)
	a, _ = step(t, a, press("P"))
	if !a.pr.reposLoading || a.pr.loading {
		t.Fatal("opening the hub must load repositories before any pull request")
	}
	a = feed(t, a, repositoriesLoadedMsg{repos: []models.Repository{{ID: "repo-1", Name: "storefront-api"}}})
	if !a.pr.loading || a.pr.repo.ID != "repo-1" {
		t.Fatal("a project with one repository must select it and start loading")
	}
	a, _ = step(t, a, pullRequestsLoadedMsg{seq: a.pr.seq, status: models.PullRequestActive, items: samplePullRequests()})
	a, _ = step(t, a, checksLoadedMsg{id: a.pr.checksFor, checks: sampleChecks()})
	return a
}

func TestPullRequestHubListsMineAndAllActive(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	out := ansi.Strip(a.render())
	for _, want := range []string{"Repos", "Pull requests 2", "Add retry policy", "Migrate invoice templates", "Build: storefront-ci", "Waiting for author"} {
		if !strings.Contains(out, want) {
			t.Errorf("Mine view is missing %q", want)
		}
	}
	if strings.Contains(out, "Fix null tenant id") {
		t.Error("Mine view shows a pull request the user neither wrote nor reviews")
	}
	a, msgs := step(t, a, press("]"))
	if _, reloading := find[pullRequestsLoadedMsg](msgs); reloading || a.pr.loading {
		t.Error("Active shares the loaded list and must not reload")
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Fix null tenant id") || !strings.Contains(out, "Pull requests 3") {
		t.Error("Active view must list every active pull request")
	}
	a, _ = step(t, a, press("]"))
	if !a.pr.loading || a.pr.panel.Selected() != nil {
		t.Error("Completed must reload and clear the stale list")
	}
	if a = feed(t, a, press("4")); a.pr.pivot != pullRequestsAbandoned {
		t.Error("4 must select Abandoned")
	}
	if a = feed(t, a, press("1")); a.pr.pivot != pullRequestsMine {
		t.Error("1 must select Mine")
	}
	if a = feed(t, a, special(tea.KeyTab)); a.pr.pivot != pullRequestsActive {
		t.Error("Tab must move to the next view")
	}
	if a = feed(t, a, press("h")); a.pr.pivot != pullRequestsMine {
		t.Error("h must move to the previous view")
	}
}

func TestPullRequestHubIgnoresStaleLoadsAndLeavesWithHubKeys(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	a, _ = step(t, a, pullRequestsLoadedMsg{seq: a.pr.seq - 1, status: models.PullRequestActive})
	if a.pr.panel.Selected() == nil {
		t.Fatal("a stale response replaced the list")
	}
	a, _ = step(t, a, press("j"))
	if got := a.pr.panel.Selected().ID; got != 4809 {
		t.Fatalf("selected %d after moving down, want 4809", got)
	}
	if a.pr.checksLoaded || a.pr.checksFor != 4809 {
		t.Error("moving the cursor must drop the previous pull request's policies")
	}
	a, msgs := step(t, a, press("o"))
	if _, ok := find[panels.BrowsePullRequestMsg](msgs); !ok {
		t.Error("o must ask to open the pull request in the browser")
	}
	a, _ = step(t, a, press("L"))
	if a.hub != panels.HubBacklog {
		t.Error("L must return to the backlog")
	}
}

func TestPullRequestHubFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {160, 45}, {139, 40}, {110, 30}, {90, 26}, {70, 22}, {50, 14}} {
		a := openPullRequestHub(t, sz[0], sz[1])
		assertFits(t, a.render(), sz[0], sz[1])
	}
}

func sampleThreads() []models.PullRequestThread {
	now := time.Now()
	return []models.PullRequestThread{
		{ID: 3, Date: now.Add(-time.Hour), Comments: []models.PullRequestComment{{ID: 1, Content: "Marco Lima voted 10", System: true}}},
		{ID: 2, Status: models.ThreadActive, FilePath: "/internal/webhook/client.go", Line: 5, Date: now.Add(-2 * time.Hour),
			Comments: []models.PullRequestComment{
				{ID: 1, AuthorName: "Marco Lima", Content: "Should the jitter be seeded per client?", Date: now.Add(-2 * time.Hour)},
				{ID: 2, ParentID: 1, AuthorName: "Sam Kowal", Content: "Good catch, pushed a fix.", Date: now.Add(-time.Hour)},
			}},
		{ID: 1, Status: models.ThreadFixed, Date: now.Add(-24 * time.Hour),
			Comments: []models.PullRequestComment{{ID: 1, AuthorName: "Priya Nair", Content: "Can we link the incident?", Date: now.Add(-24 * time.Hour)}}},
	}
}

func sampleChanges() models.PullRequestChanges {
	return models.PullRequestChanges{Iteration: 2, BaseCommit: "base", HeadCommit: "head", Files: []models.PullRequestFile{
		{Path: "/docs/webhooks.md", Kind: models.ChangeAdd},
		{Path: "/internal/webhook/client.go", Kind: models.ChangeEdit},
		{Path: "/internal/webhook/replay.go", Kind: models.ChangeDelete},
	}}
}

const (
	oldClient = "package webhook\n\nfunc (c *Client) Deliver(e Event) error {\n\tresp, err := c.http.Do(req)\n\tif err != nil {\n\t\treturn err\n\t}\n\treturn c.record(e, resp)\n}\n"
	newClient = "package webhook\n\nfunc (c *Client) Deliver(e Event) error {\n\tvar resp *http.Response\n\tfor attempt := 1; attempt <= c.maxAttempts; attempt++ {\n\t\tresp, err = c.http.Do(req)\n\t}\n\treturn c.record(e, resp)\n}\n"
)

func feed(t testing.TB, a App, msgs ...tea.Msg) App {
	t.Helper()
	var m tea.Model = a
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	return m.(App)
}

func openReview(t testing.TB, width, height int) App {
	t.Helper()
	a := openPullRequestHub(t, width, height)
	pr := *a.pr.panel.Selected()
	return feed(t, a,
		panels.ReviewPullRequestMsg{PullRequest: pr},
		threadsLoadedMsg{id: pr.ID, threads: sampleThreads()},
		checksLoadedMsg{id: pr.ID, checks: sampleChecks()},
		linkedWorkItemsLoadedMsg{id: pr.ID, items: sampleWorkItems()[:1]},
		changesLoadedMsg{id: pr.ID, changes: sampleChanges()},
	)
}

func openFiles(t testing.TB, width, height int) App {
	t.Helper()
	a := feed(t, openReview(t, width, height), press("]"), press("J"))
	return feed(t, a, fileLoadedMsg{id: 4821, path: "/internal/webhook/client.go", oldText: oldClient, newText: newClient})
}

func TestReviewOverviewShowsDescriptionPoliciesReviewersAndThreads(t *testing.T) {
	a := openReview(t, 160, 45)
	out := ansi.Strip(a.render())
	for _, want := range []string{
		"4821 Add retry policy", "Sam Kowal proposes to merge feature/webhook-retry into main", "1 active comments",
		"BLOCKED 1 required not met", "Exponential backoff", "Build: storefront-ci", "sonarqube/quality-gate",
		"Dana Reyes you", "Checkout: allow purchase", "/internal/webhook/client.go:5", "Should the jitter be seeded",
		"Good catch", "RESOLVED", "Marco Lima voted 10", "Files 3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("overview is missing %q", want)
		}
	}
}

func TestReviewThreadActionsAskForTheRightChange(t *testing.T) {
	a := openReview(t, 160, 45)
	a, msgs := step(t, a, press("s"))
	if len(msgs) != 0 {
		t.Fatal("resolving needs a selected thread")
	}
	a, _ = step(t, a, press("j"))
	a, msgs = step(t, a, press("s"))
	resolve, ok := find[panels.ResolvePullRequestThreadMsg](msgs)
	if !ok || resolve.Thread.ID != 2 || resolve.Status != models.ThreadFixed {
		t.Fatalf("got %+v, want thread 2 resolved", resolve)
	}
	a = feed(t, a, press("j"), press("j"))
	a, msgs = step(t, a, press("s"))
	if reopen, _ := find[panels.ResolvePullRequestThreadMsg](msgs); reopen.Thread.ID != 1 || reopen.Status != models.ThreadActive {
		t.Fatalf("got %+v, want thread 1 reopened", reopen)
	}
	a, msgs = step(t, a, press("r"))
	reply, ok := find[panels.ReplyPullRequestMsg](msgs)
	if !ok {
		t.Fatal("r must ask to reply")
	}
	a = feed(t, a, reply)
	topModal[modals.PullRequestCommentModal](t, a)
	for range 2 {
		a, msgs = step(t, a, special(tea.KeyEscape))
		a = feed(t, a, msgs...)
	}
	if a.viewMode != ViewMain || !a.inPullRequests() {
		t.Fatal("esc must close the dialog and then return to the list")
	}
}

func TestReviewVoteSendsTheChosenVote(t *testing.T) {
	a := openReview(t, 160, 45)
	a, msgs := step(t, a, press("a"))
	vote, ok := find[panels.VotePullRequestMsg](msgs)
	if !ok {
		t.Fatal("a must ask to vote")
	}
	a = feed(t, a, vote)
	topModal[modals.ChoiceModal](t, a)
	assertFits(t, a.render(), 160, 45)
	a, _ = step(t, a, special(tea.KeyDown))
	a, _ = step(t, a, special(tea.KeyDown))
	a, msgs = step(t, a, special(tea.KeyEnter))
	request, ok := find[modals.PullRequestVoteRequestMsg](msgs)
	if !ok || request.Vote != models.VoteWaitingForAuthor || request.PullRequest.ID != 4821 {
		t.Fatalf("got %+v, want wait-for-author on 4821", request)
	}
	a = feed(t, a, request)
	if a.modals.open() || !a.loading {
		t.Fatal("submitting must close the dialog and start saving")
	}
}

func TestReviewVoteNeedsAKnownIdentity(t *testing.T) {
	a := newTestApp(t, 160, 45)
	a = feed(t, a, press("P"), repositoriesLoadedMsg{repos: []models.Repository{{ID: "repo-1", Name: "storefront-api"}}},
		pullRequestsLoadedMsg{seq: 1, status: models.PullRequestActive, items: samplePullRequests()})
	a, _ = step(t, a, press("a"))
	if a.modals.open() || !a.notification.IsVisible() {
		t.Fatal("voting without an identity must explain itself instead of opening the dialog")
	}
}

func TestReviewFilesShowDiffThreadsAndLineComments(t *testing.T) {
	a := openFiles(t, 160, 45)
	out := ansi.Strip(a.render())
	for _, want := range []string{
		"M internal/webhook/client.go", "+3 -3", "file 2 of 3", "@@ -1,9 +1,9 @@", "var resp *http.Response",
		"resp, err := c.http.Do(req)", "Should the jitter be seeded", "1 active", "webhooks.md", "replay.go",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("files tab is missing %q", want)
		}
	}
	a, msgs := step(t, a, press("c"))
	comment, ok := find[panels.CommentPullRequestMsg](msgs)
	if !ok || comment.Anchor == nil || comment.Anchor.LeftSide != true || comment.Anchor.Line != 4 {
		t.Fatalf("first change is deleted line 4, got %+v", comment.Anchor)
	}
	a = feed(t, a, press("j"), press("j"), press("j"))
	a, msgs = step(t, a, press("c"))
	comment, _ = find[panels.CommentPullRequestMsg](msgs)
	if comment.Anchor == nil || comment.Anchor.LeftSide || comment.Anchor.Line != 4 || comment.Anchor.FilePath != "/internal/webhook/client.go" {
		t.Fatalf("want a comment on new line 4, got %+v", comment.Anchor)
	}
	a = feed(t, a, press("j"))
	a, msgs = step(t, a, press("s"))
	if _, ok := find[panels.ResolvePullRequestThreadMsg](msgs); ok {
		t.Fatal("s on a code line must not act on the thread below it")
	}
	a = feed(t, a, press("j"))
	a, msgs = step(t, a, press("s"))
	if resolve, ok := find[panels.ResolvePullRequestThreadMsg](msgs); !ok || resolve.Thread.ID != 2 {
		t.Fatalf("the focused comment's thread on new line 5 must be resolvable from the diff, got %+v", resolve)
	}
}

func TestReviewJumpsFromAThreadToItsLine(t *testing.T) {
	a := openReview(t, 160, 45)
	a = feed(t, a, press("j"), special(tea.KeyEnter))
	if a.pr.view.Tab() != panels.PullRequestFiles {
		t.Fatal("enter on a file thread must open the files tab")
	}
	file, _, ok := a.pr.view.WantedFile()
	if ok {
		t.Fatalf("the file %q was already requested by the jump", file.Path)
	}
	a = feed(t, a, fileLoadedMsg{id: 4821, path: "/internal/webhook/client.go", oldText: oldClient, newText: newClient})
	a, msgs := step(t, a, press("c"))
	if comment, _ := find[panels.CommentPullRequestMsg](msgs); comment.Anchor == nil || comment.Anchor.Line != 5 {
		t.Fatalf("cursor should sit on the thread's line 5, got %+v", comment.Anchor)
	}
}

func TestReviewFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {160, 45}, {120, 32}, {100, 30}, {80, 24}, {50, 14}} {
		overview := openReview(t, sz[0], sz[1])
		assertFits(t, overview.render(), sz[0], sz[1])
		files := openFiles(t, sz[0], sz[1])
		assertFits(t, files.render(), sz[0], sz[1])
		tree := feed(t, files, special(tea.KeyTab))
		assertFits(t, tree.render(), sz[0], sz[1])
	}
}

func twoRepositories() []models.Repository {
	return []models.Repository{{ID: "repo-1", Name: "storefront-api"}, {ID: "repo-2", Name: "storefront-web"}}
}

func TestPullRequestHubAsksForARepositoryInsteadOfListingTheProject(t *testing.T) {
	a := newTestAppAs(t, 160, 45, reviewer)
	a = feed(t, a, press("P"), repositoriesLoadedMsg{repos: twoRepositories()})
	if a.pr.loading || a.pr.seq != 0 {
		t.Fatal("no pull request may load before a repository is chosen")
	}
	topModal[modals.RepositoryPickerModal](t, a)
	out := ansi.Strip(a.render())
	for _, want := range []string{"Repo: choose", "Choose a repository", "storefront-web", "All repositories"} {
		if !strings.Contains(out, want) {
			t.Errorf("first run is missing %q", want)
		}
	}
	a, _ = step(t, a, special(tea.KeyDown))
	a, msgs := step(t, a, special(tea.KeyEnter))
	chosen, ok := find[modals.RepositoryChosenMsg](msgs)
	if !ok || chosen.Repository.ID != "repo-2" {
		t.Fatalf("got %+v, want storefront-web", chosen)
	}
	a = feed(t, a, chosen)
	if a.modals.open() || !a.pr.loading || a.savedState().Repository != "storefront-web" {
		t.Fatal("choosing must close the picker, load that repository and remember it")
	}
	a = feed(t, a, pullRequestsLoadedMsg{seq: a.pr.seq, status: models.PullRequestActive, items: samplePullRequests()[:1]})
	out = ansi.Strip(a.render())
	if !strings.Contains(out, "Pull requests / storefront-web") || !strings.Contains(out, "storefront-web ▾") {
		t.Errorf("the chosen repository must show in the breadcrumb and the header:\n%s", out)
	}
	if strings.Contains(out, "storefront-api · Sam Kowal") {
		t.Error("rows must not repeat the repository name when one is selected")
	}
}

func TestPullRequestHubRestoresTheSavedRepository(t *testing.T) {
	a := newTestAppAs(t, 160, 45, reviewer)
	a.pr.savedRepo = "storefront-web"
	a = feed(t, a, press("P"), repositoriesLoadedMsg{repos: twoRepositories()})
	if a.modals.open() || a.pr.repo.ID != "repo-2" || !a.pr.loading {
		t.Fatal("a saved repository must be selected without asking")
	}
	b := newTestAppAs(t, 160, 45, reviewer)
	b.pr.savedRepo = models.AllRepositories
	b = feed(t, b, press("P"), repositoriesLoadedMsg{repos: twoRepositories()})
	if b.modals.open() || !b.pr.allRepos || !b.pr.loading {
		t.Fatal("a saved choice of all repositories must be kept")
	}
	b = feed(t, b, pullRequestsLoadedMsg{seq: b.pr.seq, status: models.PullRequestActive, items: samplePullRequests()})
	if out := ansi.Strip(b.render()); !strings.Contains(out, "storefront-api · Sam Kowal") || !strings.Contains(out, "all ▾") {
		t.Error("the whole-project list must name each row's repository")
	}
}

func TestRepositoryPickerFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {139, 40}, {90, 26}, {70, 22}, {50, 14}} {
		a := openPullRequestHub(t, sz[0], sz[1])
		a = feed(t, a, press("r"))
		topModal[modals.RepositoryPickerModal](t, a)
		assertFits(t, a.render(), sz[0], sz[1])
	}
}

func threeRepositories() []models.Repository {
	return append(twoRepositories(), models.Repository{ID: "REPO-3", Name: "tools"})
}

func pickerRows(a App) string {
	out := ansi.Strip(a.render())
	return out[strings.Index(out, "Type to filter"):]
}

func TestRepositoryPickerListsFavoritesFirst(t *testing.T) {
	a := newTestAppAs(t, 160, 45, reviewer)
	a = feed(t, a, press("P"), repositoriesLoadedMsg{repos: threeRepositories(), starred: []string{"repo-3"}})
	rows := pickerRows(a)
	if tools, api := strings.Index(rows, "tools"), strings.Index(rows, "storefront-api"); tools < 0 || tools > api {
		t.Fatalf("a repository starred in Azure DevOps must come first:\n%s", rows)
	}
	if !strings.Contains(rows, "starred in Azure DevOps") {
		t.Error("the picker must say where the star comes from")
	}
	a = feed(t, a, special(tea.KeyDown), special(tea.KeyDown))
	a, msgs := step(t, a, ctrl('t'))
	starred, ok := find[modals.RepositoryFavoriteMsg](msgs)
	if !ok || starred.Repository.ID != "repo-2" || !starred.Favorite {
		t.Fatalf("got %+v, want storefront-web starred", starred)
	}
	a = feed(t, a, starred)
	if got := a.savedState().FavoriteRepositories; len(got) != 1 || got[0] != "repo-2" {
		t.Fatalf("saved favorites %v", got)
	}
	rows = pickerRows(a)
	if web, api := strings.Index(rows, "storefront-web"), strings.Index(rows, "storefront-api"); web > api {
		t.Fatalf("a local favorite must move above the rest:\n%s", rows)
	}
	a, msgs = step(t, a, ctrl('t'))
	if again, _ := find[modals.RepositoryFavoriteMsg](msgs); again.Repository.ID != "repo-2" || again.Favorite {
		t.Fatalf("the cursor must stay on the repository so a second press unstars it, got %+v", again)
	}
	a = feed(t, a, msgs...)
	if len(a.savedState().FavoriteRepositories) != 0 {
		t.Fatal("unstarring must be saved")
	}
	assertFits(t, a.render(), 160, 45)
}

func TestReviewSelectsSeveralLinesForOneComment(t *testing.T) {
	a := openFiles(t, 160, 45)
	a = feed(t, a, press("j"), press("j"), press("j"), press("v"), press("j"), press("j"))
	status := ansi.Strip(a.render())
	if !strings.Contains(status, "SELECT 3 LINES") || !strings.Contains(status, "comment on 3 lines") {
		t.Fatalf("the status bar must show the selection:\n%s", status[len(status)-200:])
	}
	a, msgs := step(t, a, press("c"))
	comment, ok := find[panels.CommentPullRequestMsg](msgs)
	if !ok || comment.Anchor == nil {
		t.Fatal("c must ask for a comment on the selection")
	}
	want := models.ThreadAnchor{FilePath: "/internal/webhook/client.go", Line: 4, EndLine: 6, EndOffset: len("\t\tresp, err = c.http.Do(req)") + 1}
	if *comment.Anchor != want {
		t.Fatalf("got %+v, want %+v", *comment.Anchor, want)
	}
	a = feed(t, a, comment)
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Comment on lines") || !strings.Contains(out, "internal/webhook/client.go:4-6") {
		t.Error("the dialog must name the line range")
	}
	if strings.Contains(ansi.Strip(a.render()), "SELECT") {
		t.Error("the selection must end once the comment dialog opens")
	}
}

func TestReviewSelectionStaysInItsHunkAndEscCancelsIt(t *testing.T) {
	a := openFiles(t, 160, 45)
	a = feed(t, a, press("v"), press("G"))
	if !strings.Contains(ansi.Strip(a.render()), "SELECT 9 LINES") {
		t.Fatalf("selection from deleted line 4 to the end of the hunk should be 9 lines")
	}
	a, msgs := step(t, a, special(tea.KeyEscape))
	if _, closed := find[panels.ClosePullRequestViewMsg](msgs); closed || a.viewMode != ViewPullRequest {
		t.Fatal("esc must cancel the selection before it leaves the review")
	}
	if strings.Contains(ansi.Strip(a.render()), "SELECT") {
		t.Fatal("esc must clear the selection")
	}
	a = feed(t, a, press("g"), press("j"), press("j"), press("j"), press("j"), press("v"), press("j"), press("j"))
	a, msgs = step(t, a, press("c"))
	if comment, _ := find[panels.CommentPullRequestMsg](msgs); comment.Anchor == nil || !comment.Anchor.LeftSide || comment.Anchor.Line != 4 || comment.Anchor.EndLine != 6 {
		t.Fatalf("three deleted lines must anchor on the old side, got %+v", comment.Anchor)
	}
}

func TestReviewShowsARangeThreadUnderItsLastLine(t *testing.T) {
	a := openFiles(t, 160, 45)
	threads := sampleThreads()
	threads[1].Line, threads[1].EndLine = 4, 6
	a = feed(t, a, threadsLoadedMsg{id: 4821, threads: threads})
	out := ansi.Strip(a.render())
	thread, last := strings.Index(out, "Lines 4-6"), strings.Index(out, "resp, err = c.http.Do(req)\n")
	if thread < 0 || strings.Index(out, "c.http.Do(req)") > thread {
		t.Fatalf("the range thread must sit under new line 6 (thread at %d, line at %d)", thread, last)
	}
	if strings.Count(out, "│     ") < 2 {
		t.Error("the lines a range thread covers must be marked in the gutter")
	}
	a = feed(t, a, press("["))
	if !strings.Contains(ansi.Strip(a.render()), "/internal/webhook/client.go:4-6") {
		t.Error("the overview must name the range")
	}
}

func TestReviewCommentsCanMentionTeamMembers(t *testing.T) {
	a := openReview(t, 160, 45)
	a.teamMembers = []models.TeamMember{{ID: "11111111-2222-3333-4444-555555555555", DisplayName: "Dana Reyes"}, {ID: "u9", DisplayName: "Lena Ortiz"}}
	a = feed(t, a, panels.CommentPullRequestMsg{PullRequest: *a.pr.view.PullRequest()})
	for _, r := range "Ping @Le" {
		a = feed(t, a, press(string(r)))
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Mention") || !strings.Contains(out, "Lena Ortiz") {
		t.Fatalf("typing @ must open the mention picker:\n%s", out)
	}
	a = feed(t, a, special(tea.KeyEnter))
	a, msgs := step(t, a, ctrl('s'))
	request, ok := find[modals.PullRequestCommentRequestMsg](msgs)
	if !ok || request.Text != "Ping @<u9>" {
		t.Fatalf("got %q, want the mention sent as an identity reference", request.Text)
	}
}

func TestReviewShowsMentionsByName(t *testing.T) {
	a := openReview(t, 160, 45)
	a.pr.view.SetMembers([]models.TeamMember{{ID: "11111111-2222-3333-4444-555555555555", DisplayName: "Dana Reyes"}})
	threads := sampleThreads()
	threads[1].Comments[0].Content = "Can @<11111111-2222-3333-4444-555555555555> and @<99999999-2222-3333-4444-555555555555> check this?"
	a = feed(t, a, threadsLoadedMsg{id: 4821, threads: threads})
	out := ansi.Strip(a.render())
	if !strings.Contains(out, "Can @Dana Reyes and @user check this?") || strings.Contains(out, "11111111") {
		t.Fatalf("mentions must read as names:\n%s", out)
	}
}

func TestReviewOpensLinkedWorkItemsAndComesBack(t *testing.T) {
	a := openReview(t, 160, 45)
	a, msgs := step(t, a, press("w"))
	linked, ok := find[panels.LinkedWorkItemsMsg](msgs)
	if !ok || len(linked.Items) != 1 {
		t.Fatalf("w must ask for the linked work items, got %+v", linked)
	}
	a = feed(t, a, linked)
	if a.viewMode != ViewDetail || a.detailView.Item().ID != 1043 {
		t.Fatal("a single linked work item must open in the work item view")
	}
	a, msgs = step(t, a, special(tea.KeyEscape))
	a = feed(t, a, msgs...)
	if !a.reviewing() {
		t.Fatal("closing the work item must return to the review")
	}
	linked.Items = sampleWorkItems()[:3]
	a = feed(t, a, linked)
	topModal[modals.ChoiceModal](t, a)
	assertFits(t, a.render(), 160, 45)
	a = feed(t, a, special(tea.KeyDown))
	a, msgs = step(t, a, special(tea.KeyEnter))
	chosen, ok := find[modals.WorkItemChosenMsg](msgs)
	if !ok || chosen.Item.ID != 1051 {
		t.Fatalf("got %+v, want the second linked item", chosen)
	}
	a = feed(t, a, chosen)
	if a.viewMode != ViewDetail || a.modals.open() {
		t.Fatal("choosing must open the work item")
	}
}

func ownThreads() []models.PullRequestThread {
	threads := sampleThreads()
	threads[1].Comments[1].AuthorID, threads[1].Comments[1].AuthorName = "me-id", "Dana Reyes"
	return threads
}

func TestReviewEditsAndDeletesOnlyOwnComments(t *testing.T) {
	a := openReview(t, 160, 45)
	a = feed(t, a, threadsLoadedMsg{id: 4821, threads: ownThreads()}, press("j"))
	a, msgs := step(t, a, press("e"))
	if _, ok := find[panels.CommentReadOnlyMsg](msgs); !ok {
		t.Fatal("editing someone else's comment must be refused")
	}
	a = feed(t, a, msgs...)
	if a.modals.open() || !a.notification.IsVisible() {
		t.Fatal("a refused edit must explain itself, not open a dialog")
	}
	a = feed(t, a, press("j"))
	if out := ansi.Strip(a.render()); !strings.Contains(out, "╭─ DR Dana Reyes (you)") || !strings.Contains(out, "╰─ r reply  e edit  d delete") {
		t.Fatalf("the focused own comment must be marked and offer edit:\n%s", out)
	}
	a, msgs = step(t, a, press("e"))
	edit, ok := find[panels.EditPullRequestCommentMsg](msgs)
	if !ok || edit.Thread.ID != 2 || edit.Comment.ID != 2 {
		t.Fatalf("got %+v, want comment 2 of thread 2", edit)
	}
	a = feed(t, a, edit)
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Edit comment") || !strings.Contains(out, "Good catch, pushed a fix.") {
		t.Fatal("the edit dialog must start with the comment's text")
	}
	a = feed(t, a, press("!"))
	a, msgs = step(t, a, ctrl('s'))
	request, ok := find[modals.PullRequestCommentRequestMsg](msgs)
	if !ok || request.CommentID != 2 || request.ThreadID != 2 || request.Text != "Good catch, pushed a fix.!" {
		t.Fatalf("got %+v", request)
	}
	a = feed(t, a, request)
	if a.modals.open() || !a.loading {
		t.Fatal("saving must close the dialog and start the update")
	}
	a, msgs = step(t, a, press("d"))
	remove, ok := find[panels.DeletePullRequestCommentMsg](msgs)
	if !ok || remove.Comment.ID != 2 {
		t.Fatalf("got %+v, want comment 2", remove)
	}
	a = feed(t, a, remove)
	topModal[modals.ConfirmModal](t, a)
	assertFits(t, a.render(), 160, 45)
	a, msgs = step(t, a, press("y"))
	confirmed, ok := find[modals.PullRequestCommentDeleteRequestMsg](msgs)
	if !ok || confirmed.ThreadID != 2 || confirmed.CommentID != 2 {
		t.Fatalf("got %+v", confirmed)
	}
}

func TestReviewEditsOwnCommentFromTheDiff(t *testing.T) {
	a := openFiles(t, 160, 45)
	a = feed(t, a, threadsLoadedMsg{id: 4821, threads: ownThreads()}, press("j"), press("j"), press("j"), press("j"), press("j"))
	if out := ansi.Strip(a.render()); !strings.Contains(out, "╭─ ML Marco Lima") || !strings.Contains(out, "╰─ r reply  + like") {
		t.Fatalf("j must step from the code line into the first comment and offer a like:\n%s", out)
	}
	a, msgs := step(t, a, press("e"))
	if _, ok := find[panels.CommentReadOnlyMsg](msgs); !ok {
		t.Fatal("e on someone else's comment in the diff must be refused")
	}
	a = feed(t, a, press("j"))
	a, msgs = step(t, a, press("e"))
	edit, ok := find[panels.EditPullRequestCommentMsg](msgs)
	if !ok || edit.Comment.ID != 2 {
		t.Fatalf("e on the focused own comment in the diff must edit it, got %+v", edit)
	}
	a, msgs = step(t, a, press("d"))
	if remove, ok := find[panels.DeletePullRequestCommentMsg](msgs); !ok || remove.Comment.ID != 2 {
		t.Fatalf("d on the focused own comment must delete it, got %+v", remove)
	}
	a = feed(t, a, press("j"))
	a, msgs = step(t, a, press("c"))
	if comment, _ := find[panels.CommentPullRequestMsg](msgs); comment.Anchor == nil || comment.Anchor.Line != 6 {
		t.Fatalf("j past the last comment must land on the next code line, got %+v", comment.Anchor)
	}
	a = feed(t, a, press("k"))
	if _, c, ok := a.pr.view.FocusedComment(); !ok || c.ID != 2 {
		t.Fatalf("k from the next line must come back to the thread's last comment, got %+v", c)
	}
}

func TestReviewLikesOnlyOtherPeoplesComments(t *testing.T) {
	a := openReview(t, 160, 45)
	threads := ownThreads()
	threads[1].Comments[0].Likes = []models.Identity{{ID: "u4"}}
	a = feed(t, a, threadsLoadedMsg{id: 4821, threads: threads}, press("j"))
	if out := ansi.Strip(a.render()); !strings.Contains(out, "♥ 1 ─╮") || strings.Contains(out, "· you ─╮") || !strings.Contains(out, "+ like") || strings.Contains(out, "e edit") {
		t.Fatalf("someone else's comment must show its likes and offer like, not edit:\n%s", out)
	}
	a, msgs := step(t, a, press("+"))
	like, ok := find[panels.LikePullRequestCommentMsg](msgs)
	if !ok || !like.Like || like.Thread.ID != 2 || like.Comment.ID != 1 {
		t.Fatalf("got %+v, want a like of comment 1 in thread 2", like)
	}
	a = feed(t, a, like)
	if !a.loading {
		t.Fatal("a like must start the update")
	}
	threads[1].Comments[0].Likes = append(threads[1].Comments[0].Likes, models.Identity{ID: "me-id"})
	a = feed(t, a, threadsLoadedMsg{id: 4821, threads: threads})
	if out := ansi.Strip(a.render()); !strings.Contains(out, "+ unlike") || !strings.Contains(out, "♥ 2 · you ─╮") {
		t.Fatalf("a liked comment must offer unlike:\n%s", out)
	}
	_, msgs = step(t, a, press("+"))
	if unlike, ok := find[panels.LikePullRequestCommentMsg](msgs); !ok || unlike.Like {
		t.Fatalf("got %+v, want the like removed", unlike)
	}
	a = feed(t, a, press("j"))
	_, msgs = step(t, a, press("+"))
	if _, ok := find[panels.LikePullRequestCommentMsg](msgs); ok {
		t.Fatal("the user's own comment must not be liked")
	}
}

func TestReviewRepliesToTheFocusedComment(t *testing.T) {
	a := feed(t, openReview(t, 160, 45), press("j"), press("j"))
	a, msgs := step(t, a, press("r"))
	reply, ok := find[panels.ReplyPullRequestMsg](msgs)
	if !ok || reply.Comment.ID != 2 {
		t.Fatalf("got %+v, want a reply to comment 2", reply)
	}
	a = feed(t, a, reply)
	if out := ansi.Strip(a.render()); !strings.Contains(out, "To Sam Kowal") {
		t.Fatal("the reply dialog must name the focused comment's author")
	}
	a = feed(t, a, press("o"), press("k"))
	_, msgs = step(t, a, ctrl('s'))
	if req, ok := find[modals.PullRequestCommentRequestMsg](msgs); !ok || req.ParentID != 2 || req.ThreadID != 2 {
		t.Fatalf("got %+v, want a reply under comment 2 of thread 2", req)
	}
}

func TestReviewResolvedThreadsFoldUntilExpanded(t *testing.T) {
	a := openReview(t, 160, 45)
	out := ansi.Strip(a.render())
	if strings.Contains(out, "Can we link the incident?") || !strings.Contains(out, "General comment · 1 comment · last by Priya Nair") {
		t.Fatalf("a resolved thread must fold to one line:\n%s", out)
	}
	if strings.Contains(out, "Thread ") {
		t.Fatal("threads must be labelled by their anchor, not \"Thread\"")
	}
	a = feed(t, a, press("j"), press("j"), press("j"))
	if out = ansi.Strip(a.render()); !strings.Contains(out, "Enter expand") || !strings.Contains(out, "s reopen") {
		t.Fatalf("a focused folded thread must offer expand and reopen:\n%s", out)
	}
	_, msgs := step(t, a, press("e"))
	if _, ok := find[panels.EditPullRequestCommentMsg](msgs); ok {
		t.Fatal("a folded thread must not edit a hidden comment")
	}
	a = feed(t, a, special(tea.KeyEnter))
	if out = ansi.Strip(a.render()); !strings.Contains(out, "Can we link the incident?") {
		t.Fatalf("enter must expand the folded thread:\n%s", out)
	}
	a = feed(t, a, special(tea.KeyEnter))
	if out = ansi.Strip(a.render()); strings.Contains(out, "Can we link the incident?") {
		t.Fatal("enter on an expanded general thread must fold it again")
	}
}

func TestReviewOpensTheFocusedThreadInTheBrowser(t *testing.T) {
	a := feed(t, openReview(t, 160, 45), press("j"))
	_, msgs := step(t, a, press("o"))
	browse, ok := find[panels.BrowsePullRequestMsg](msgs)
	if !ok || !strings.HasSuffix(browse.PullRequest.WebURL, "/pullrequest/1?discussionId=2") {
		t.Fatalf("got %+v, want a link to discussion 2", browse.PullRequest.WebURL)
	}
}

func TestReviewShowsThreadsOnLinesOutsideTheChanges(t *testing.T) {
	var oldLines []string
	for i := 1; i <= 60; i++ {
		oldLines = append(oldLines, fmt.Sprintf("line %d", i))
	}
	newLines := append([]string{}, oldLines...)
	newLines[1] = "changed"
	now := time.Now()
	threads := []models.PullRequestThread{
		{ID: 7, Status: models.ThreadFixed, FilePath: "internal/webhook/Client.go", Line: 40, Date: now,
			Comments: []models.PullRequestComment{{ID: 1, AuthorName: "Marco Lima", Content: "Resolved note far from the change", Date: now}}},
		{ID: 8, Status: "closed", FilePath: "/internal/webhook/client.go", Line: 900, Date: now,
			Comments: []models.PullRequestComment{{ID: 1, AuthorName: "Priya Nair", Content: "On a line that no longer exists", Date: now}}},
		{ID: 9, Status: models.ThreadActive, FilePath: "/internal/webhook/client.go", Date: now,
			Comments: []models.PullRequestComment{{ID: 1, AuthorName: "Sam Kowal", Content: "A comment on the whole file", Date: now}}},
	}
	a := feed(t, openReview(t, 160, 60), press("]"), press("J"), threadsLoadedMsg{id: 4821, threads: threads},
		fileLoadedMsg{id: 4821, path: "/internal/webhook/client.go", oldText: strings.Join(oldLines, "\n"), newText: strings.Join(newLines, "\n")})
	out := ansi.Strip(a.render())
	for _, want := range []string{"Line 40 · 1 comment · last by Marco Lima", "RESOLVED", "line 40", "CLOSED", "last by Priya Nair", "A comment on the whole file", "+1 -1"} {
		if !strings.Contains(out, want) {
			t.Errorf("files tab is missing %q", want)
		}
	}
	if strings.Index(out, "A comment on the whole file") > strings.Index(out, "@@ -1,") {
		t.Error("threads without a line in the diff must be listed above it, not after the last line")
	}
	if strings.Index(out, "line 40") > strings.Index(out, "last by Marco Lima") {
		t.Error("a thread outside the changes must sit under its own line")
	}
	a = feed(t, a, press("g"), press("k"), press("k"), special(tea.KeyEnter))
	if out = ansi.Strip(a.render()); !strings.Contains(out, "A comment on the whole file") || !strings.Contains(out, "On a line that no longer exists") {
		t.Fatalf("enter on a folded thread above the diff must expand it:\n%s", out)
	}
}

func TestReviewReloadsThreadsTrackedToTheShownIteration(t *testing.T) {
	a := openPullRequestHub(t, 160, 45)
	pr := *a.pr.panel.Selected()
	a = feed(t, a, panels.ReviewPullRequestMsg{PullRequest: pr})
	if a.pr.view.Iteration() != 0 {
		t.Fatal("no iteration is known before the changes load")
	}
	a = feed(t, a, changesLoadedMsg{id: pr.ID, changes: sampleChanges()})
	if a.pr.view.Iteration() != 2 {
		t.Fatalf("got iteration %d, want 2", a.pr.view.Iteration())
	}
}

func TestReviewFileListFlagsFilesWithOnlyClosedComments(t *testing.T) {
	now := time.Now()
	threads := []models.PullRequestThread{
		{ID: 7, Status: models.ThreadFixed, FilePath: "/docs/webhooks.md", Line: 1, Date: now,
			Comments: []models.PullRequestComment{{ID: 1, AuthorName: "Marco Lima", Content: "Typo", Date: now}}},
		{ID: 8, Status: "closed", FilePath: "/docs/webhooks.md", Line: 2, Date: now,
			Comments: []models.PullRequestComment{{ID: 1, AuthorName: "Marco Lima", Content: "Reword", Date: now}}},
	}
	a := feed(t, openFiles(t, 160, 45), threadsLoadedMsg{id: 4821, threads: append(threads, sampleThreads()...)})
	out := ansi.Strip(a.render())
	if !strings.Contains(out, "2 comments") || !strings.Contains(out, "1 active") {
		t.Fatalf("the file list must flag closed comments as well as active ones:\n%s", out)
	}
}

func TestReviewKeepsTheNextLinesThreadInView(t *testing.T) {
	var lines []string
	for i := 1; i <= 80; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	now := time.Now()
	thread := models.PullRequestThread{ID: 7, Status: models.ThreadActive, FilePath: "/internal/webhook/client.go", Line: 21, Date: now,
		Comments: []models.PullRequestComment{
			{ID: 1, AuthorName: "Marco Lima", Content: "First comment", Date: now},
			{ID: 2, ParentID: 1, AuthorName: "Sam Kowal", Content: "Last reply in the thread", Date: now},
		}}
	a := feed(t, openReview(t, 120, 30), press("]"), press("J"), threadsLoadedMsg{id: 4821, threads: []models.PullRequestThread{thread}},
		fileLoadedMsg{id: 4821, path: "/internal/webhook/client.go", newText: strings.Join(lines, "\n")})
	for range 19 {
		a = feed(t, a, press("j"))
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Last reply in the thread") {
		t.Fatalf("with the cursor one line above, the whole thread must be on screen:\n%s", out)
	}
}

func TestQuickSearchOpensAPullRequestFromAnyHub(t *testing.T) {
	a := newTestAppAs(t, 160, 45, reviewer)
	pr := samplePullRequests()[1]
	a = feed(t, a, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}, modals.QuickSearchSuccessMsg{PullRequest: &pr})
	if !a.reviewing() || a.modals.open() || a.pr.view.PullRequest().ID != pr.ID {
		t.Fatal("a pull request found by quick search must open its review")
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "4817 Fix null tenant id in audit export") {
		t.Fatalf("the review must show the searched pull request:\n%s", out)
	}
}

func TestQuickSearchAsksWhenANumberMatchesBoth(t *testing.T) {
	a := newTestAppAs(t, 160, 45, reviewer)
	pr := samplePullRequests()[0]
	item := sampleWorkItems()[0]
	a = feed(t, a, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}, modals.QuickSearchSuccessMsg{Item: &item, PullRequest: &pr})
	topModal[modals.QuickSearchModal](t, a)
	if a.reviewing() || a.viewMode != ViewMain {
		t.Fatal("two matches must wait for the user's choice")
	}
	a, msgs := step(t, a, special(tea.KeyEnter))
	a = feed(t, a, msgs...)
	if !a.reviewing() || a.modals.open() {
		t.Fatal("choosing the pull request must open its review")
	}
}

func lockfile(lines int) string {
	var out []string
	for i := range lines {
		out = append(out, fmt.Sprintf(`    "node_modules/pkg-%d": { "version": "1.0.%d" },`, i, i))
	}
	return strings.Join(out, "\n")
}

func TestReviewShowsLongFilesAsTooLargeUntilAskedFor(t *testing.T) {
	a := feed(t, openReview(t, 160, 45), press("]"),
		fileLoadedMsg{id: 4821, path: "/docs/webhooks.md", newText: lockfile(4000)})
	out := ansi.Strip(a.render())
	for _, want := range []string{"This file is too large to show", "4000 lines; the limit is 3000 (diff.max_file_lines", "Enter load diff", "not rendered", "too large"} {
		if !strings.Contains(out, want) {
			t.Errorf("files tab is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "pkg-10") {
		t.Fatal("a file over the line limit must not be rendered")
	}
	a = feed(t, a, special(tea.KeyEnter))
	if out = ansi.Strip(a.render()); !strings.Contains(out, `"node_modules/pkg-0"`) || strings.Contains(out, "This file is too large") {
		t.Fatalf("enter must load the diff anyway:\n%s", out)
	}
}

func TestReviewShowsLargeDiffsAsTooLarge(t *testing.T) {
	a := feed(t, openReview(t, 160, 45), press("]"),
		fileLoadedMsg{id: 4821, path: "/docs/webhooks.md", newText: lockfile(1200)})
	if out := ansi.Strip(a.render()); !strings.Contains(out, "This file is too large to show") || !strings.Contains(out, "About 1200 changed lines; the limit is 1000") {
		t.Fatalf("a diff over the changed-lines limit must be collapsed:\n%s", out)
	}
}

func TestReviewShowsFilesOverTheSizeCapAsTooLarge(t *testing.T) {
	a := feed(t, openReview(t, 160, 45), press("]"),
		fileLoadedMsg{id: 4821, path: "/docs/webhooks.md", err: fmt.Errorf("loading /docs/webhooks.md: %w", models.ErrFileTooLarge)})
	out := ansi.Strip(a.render())
	if !strings.Contains(out, "This file is too large to show") || !strings.Contains(out, "Over 1024 KB") || strings.Contains(out, "load diff") {
		t.Fatalf("a file that was never downloaded can only be opened in the browser:\n%s", out)
	}
	_, msgs := step(t, a, press("o"))
	if browse, ok := find[panels.BrowsePullRequestMsg](msgs); !ok || !strings.HasSuffix(browse.PullRequest.WebURL, "?_a=files&path=%2Fdocs%2Fwebhooks.md") {
		t.Fatalf("o must open the file in the browser, got %+v", browse.PullRequest.WebURL)
	}
}

func TestReviewCollapsesFilesTheRepositoryMarksAsGenerated(t *testing.T) {
	changes := sampleChanges()
	changes.Attributes = "*.md linguist-generated=true\n"
	a := feed(t, openReview(t, 160, 45), changesLoadedMsg{id: 4821, changes: changes}, press("]"),
		fileLoadedMsg{id: 4821, path: "/docs/webhooks.md", newText: "# Webhooks\n\nRetries.\n"})
	out := ansi.Strip(a.render())
	if !strings.Contains(out, "Generated file not rendered") || !strings.Contains(out, "generated") || strings.Contains(out, "Retries.") {
		t.Fatalf("a file marked linguist-generated must be collapsed:\n%s", out)
	}
}

func TestPullRequestSearchFiltersAsYouType(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	a = feed(t, a, press("/"))
	out := ansi.Strip(a.render())
	for _, want := range []string{"SEARCH", "author:", "reviewer:", "source:", "target:", "is:draft", "server"} {
		if !strings.Contains(out, want) {
			t.Errorf("an empty search bar must suggest filters, missing %q:\n%s", want, out)
		}
	}
	a = typeText(t, a, "liquid")
	if !a.pr.panel.Searching() {
		t.Fatal("letters typed in the bar, q included, must stay in the bar")
	}
	out = ansi.Strip(a.render())
	if !strings.Contains(out, "1 of 2") || !strings.Contains(out, "Migrate invoice templates") || strings.Contains(out, "Add retry policy") {
		t.Fatalf("typing must filter the loaded list:\n%s", out)
	}
	a = feed(t, a, special(tea.KeyEscape))
	if out = ansi.Strip(a.render()); !strings.Contains(out, "Add retry policy") || !a.pr.panel.Searching() {
		t.Fatal("the first esc must clear the search and keep the bar open")
	}
	a = feed(t, a, special(tea.KeyEscape))
	if a.pr.panel.Searching() {
		t.Fatal("the second esc must leave the bar")
	}
}

func TestPullRequestSearchSendsFiltersToTheServer(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	a = typeText(t, feed(t, a, press("/")), "author:@me")
	a, msgs := step(t, a, special(tea.KeyEnter))
	search, ok := find[panels.SearchPullRequestsMsg](msgs)
	if !ok || search.Criteria.CreatorID != "me-id" {
		t.Fatalf("got %+v, want a server search for my pull requests", search)
	}
	a = feed(t, a, search)
	if a.pr.criteria.CreatorID != "me-id" || !a.pr.loading {
		t.Fatal("a server filter must reload the list with the criteria")
	}
	a = feed(t, a, pullRequestsLoadedMsg{seq: a.pr.seq, status: models.PullRequestActive, items: samplePullRequests()[2:]})
	if out := ansi.Strip(a.render()); !strings.Contains(out, "1 of 1 · server") {
		t.Fatalf("the bar must say the results came from the server:\n%s", out)
	}
	a = feed(t, a, press("/"))
	a, msgs = step(t, a, special(tea.KeyEscape))
	if clear, ok := find[panels.SearchPullRequestsMsg](msgs); !ok || !clear.Criteria.Empty() {
		t.Fatalf("clearing must drop the server filter, got %+v", clear)
	}
}

func TestPullRequestSearchOpensAnIDOutsideTheView(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	a = typeText(t, feed(t, a, press("/")), "4511")
	if out := ansi.Strip(a.render()); !strings.Contains(out, "!4511 isn't in this view") {
		t.Fatalf("an unknown ID must offer to open it anyway:\n%s", out)
	}
	_, msgs := step(t, a, special(tea.KeyEnter))
	if open, ok := find[panels.OpenPullRequestByIDMsg](msgs); !ok || open.ID != 4511 {
		t.Fatalf("got %+v, want pull request 4511 opened by ID", open)
	}
}

func TestPullRequestSearchCompletesReviewerNames(t *testing.T) {
	a := openPullRequestHub(t, 160, 40)
	a.teamMembers = append(a.teamMembers, models.TeamMember{ID: "pn-id", DisplayName: "Priya Nair", UniqueName: "priya@example.com"})
	a.pr.panel.SetMembers(a.teamMembers)
	a = typeText(t, feed(t, a, press("/")), "reviewer:pri")
	a = feed(t, a, special(tea.KeyTab))
	if out := ansi.Strip(a.render()); !strings.Contains(out, `reviewer:"Priya Nair"`) {
		t.Fatalf("tab must complete the reviewer's full name:\n%s", out)
	}
	_, msgs := step(t, a, special(tea.KeyEnter))
	if search, ok := find[panels.SearchPullRequestsMsg](msgs); !ok || search.Criteria.ReviewerID != "pn-id" {
		t.Fatalf("got %+v, want a server search by reviewer ID", search)
	}
}

func openPullRequestForm(t *testing.T) App {
	t.Helper()
	a := openPullRequestHub(t, 160, 50)
	a.teamMembers = []models.TeamMember{{ID: "ml-id", DisplayName: "Marco Lima"}, {ID: "pn-id", DisplayName: "Priya Nair"}}
	a, msgs := step(t, a, press("n"))
	topModal[modals.PullRequestForm](t, a)
	if req, ok := find[modals.PullRequestBranchesRequestMsg](msgs); !ok || req.RepositoryID != "repo-1" {
		t.Fatalf("opening the form must load the repository's branches, got %+v", req)
	}
	return feed(t, a, modals.PullRequestBranchesMsg{
		RepositoryID: "repo-1", Branches: []string{"feature/1043-guest-checkout", "feature/webhook-retry", "main"}, Default: "main",
	})
}

func chooseSource(t *testing.T, a App, filter string) (App, modals.PullRequestCompareRequestMsg) {
	t.Helper()
	a = typeText(t, a, filter)
	a, msgs := step(t, a, special(tea.KeyEnter))
	req, ok := find[modals.PullRequestCompareRequestMsg](msgs)
	if !ok {
		t.Fatalf("leaving the source branch must compare it with the target, got %v", msgs)
	}
	return a, req
}

func TestNewPullRequestComparesBranchesPrefillsAndCreates(t *testing.T) {
	a := openPullRequestForm(t)
	out := ansi.Strip(a.render())
	for _, want := range []string{"New pull request", "storefront-api", "Source branch", "Target branch", "main", "Required reviewers", "Ctrl+s create"} {
		if !strings.Contains(out, want) {
			t.Errorf("the form is missing %q:\n%s", want, out)
		}
	}
	a, req := chooseSource(t, a, "guest")
	if req.Source != "feature/1043-guest-checkout" || req.Target != "main" || req.RepositoryID != "repo-1" {
		t.Fatalf("got %+v", req)
	}
	if out = ansi.Strip(a.render()); !strings.Contains(out, "Comparing branches") {
		t.Fatalf("the form must say it is comparing:\n%s", out)
	}
	a = feed(t, a, modals.PullRequestComparedMsg{RepositoryID: "repo-1", Source: req.Source, Target: req.Target,
		Comparison: models.BranchComparison{Ahead: 1, Files: 7, Commits: []string{"Checkout: allow guests"}, Template: "## What changed\n"}})
	out = ansi.Strip(a.render())
	for _, want := range []string{"1 commit ahead, 0 behind main · 7 files changed", "Checkout: allow guests", "## What changed", "1043"} {
		if !strings.Contains(out, want) {
			t.Errorf("the comparison must fill the form, missing %q:\n%s", want, out)
		}
	}
	a = feed(t, a, special(tea.KeyTab), special(tea.KeyTab), special(tea.KeyTab))
	a = typeText(t, a, "marco")
	a = feed(t, a, special(tea.KeyEnter))
	a, msgs := step(t, a, ctrl('s'))
	create, ok := find[modals.PullRequestCreateRequestMsg](msgs)
	if !ok {
		t.Fatalf("ctrl+s must create the pull request, got %v", msgs)
	}
	pr := create.PullRequest
	if create.Repository.ID != "repo-1" || pr.Source != "feature/1043-guest-checkout" || pr.Target != "main" || pr.Title != "Checkout: allow guests" ||
		len(pr.WorkItems) != 1 || pr.WorkItems[0] != 1043 || len(pr.Reviewers) != 1 || pr.Reviewers[0] != (models.NewReviewer{ID: "ml-id", Required: true}) {
		t.Fatalf("got %+v", create)
	}
	m, _ := a.Update(create)
	a = m.(App)
	if !a.loading || !a.modals.open() {
		t.Fatal("creating must keep the form open until Azure DevOps answers")
	}
	created := samplePullRequests()[0]
	a = feed(t, a, pullRequestCreatedMsg{pr: created})
	if a.modals.open() || !a.reviewing() || a.pr.view.PullRequest().ID != created.ID {
		t.Fatal("a created pull request must close the form and open its review")
	}
}

func TestNewPullRequestPointsToAnOpenOneInsteadOfDuplicating(t *testing.T) {
	a, req := chooseSource(t, openPullRequestForm(t), "webhook")
	a = feed(t, a, modals.PullRequestComparedMsg{RepositoryID: "repo-1", Source: req.Source, Target: req.Target,
		Comparison: models.BranchComparison{ExistingID: 4821, ExistingTitle: "Add retry policy"}})
	if out := ansi.Strip(a.render()); !strings.Contains(out, "An active pull request already exists: !4821 Add retry policy") {
		t.Fatalf("the form must point to the open pull request:\n%s", out)
	}
	_, msgs := step(t, a, ctrl('s'))
	if open, ok := find[modals.OpenPullRequestRequestMsg](msgs); !ok || open.ID != 4821 {
		t.Fatalf("ctrl+s must open the existing pull request, got %v", msgs)
	}
}

func TestNewPullRequestRefusesBranchesWithNothingToMerge(t *testing.T) {
	a, req := chooseSource(t, openPullRequestForm(t), "guest")
	a = feed(t, a, modals.PullRequestComparedMsg{RepositoryID: "repo-1", Source: req.Source, Target: req.Target, Comparison: models.BranchComparison{Behind: 3}})
	a, msgs := step(t, a, ctrl('s'))
	if _, ok := find[modals.PullRequestCreateRequestMsg](msgs); ok {
		t.Fatal("a branch with no new commits must not be submitted")
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Nothing to merge: feature/1043-guest-checkout has no commits that aren't in main") {
		t.Fatalf("the form must explain why:\n%s", out)
	}
}

func TestNewPullRequestStartsWithTheRepositoryWhenTheHubShowsAll(t *testing.T) {
	a := openPullRequestHub(t, 160, 50)
	a.pr.repos = append(a.pr.repos, models.Repository{ID: "repo-2", Name: "web"})
	a = feed(t, a, modals.RepositoryChosenMsg{All: true})
	a, msgs := step(t, a, press("n"))
	if _, ok := find[modals.PullRequestBranchesRequestMsg](msgs); ok {
		t.Fatal("branches must wait until a repository is chosen")
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Repository") || !strings.Contains(out, "Choose a repository") {
		t.Fatalf("the form must ask for the repository first:\n%s", out)
	}
	a = typeText(t, a, "web")
	_, msgs = step(t, a, special(tea.KeyEnter))
	if req, ok := find[modals.PullRequestBranchesRequestMsg](msgs); !ok || req.RepositoryID != "repo-2" {
		t.Fatalf("choosing a repository must load its branches, got %v", msgs)
	}
}
