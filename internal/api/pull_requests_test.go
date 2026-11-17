package api

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/models"
)

type recorded struct {
	method string
	url    string
	body   string
}

func clientRouting(t *testing.T, routes map[string]string, seen *[]recorded) *Client {
	t.Helper()
	c := NewClient(&config.Config{Organization: "contoso", Project: "web", Team: "web team", PAT: "test-pat-not-real"})
	c.httpClient.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		body := ""
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			body = string(raw)
		}
		*seen = append(*seen, recorded{method: r.Method, url: r.URL.String(), body: body})
		reply, matched := `{"value":[]}`, ""
		for fragment, candidate := range routes {
			if strings.Contains(r.URL.Path, fragment) && len(fragment) > len(matched) {
				reply, matched = candidate, fragment
			}
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(reply)), Header: http.Header{}}, nil
	})
	return c
}

func TestListPullRequestsMapsBranchesReviewersAndWebURL(t *testing.T) {
	var seen []recorded
	body := `{"value":[{"pullRequestId":4821,"title":"Add retry policy","status":"active","isDraft":true,` +
		`"createdBy":{"id":"a1","displayName":"Marta Ruiz","uniqueName":"marta@example.com"},` +
		`"creationDate":"2026-11-14T09:30:00.123Z","sourceRefName":"refs/heads/feature/retry","targetRefName":"refs/heads/main",` +
		`"mergeStatus":"conflicts","repository":{"id":"r1","name":"phoenix api","project":{"id":"p1"}},` +
		`"reviewers":[{"id":"u1","displayName":"Deepak Kumar","vote":10,"isRequired":true},{"id":"g1","displayName":"Team","vote":-5,"isContainer":true}],` +
		`"labels":[{"name":"payments"}]}]}`
	prs, err := clientRouting(t, map[string]string{"/git/pullrequests": body}, &seen).ListPullRequests(models.PullRequestActive, "", models.PullRequestCriteria{})
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 1 {
		t.Fatalf("got %d pull requests, want 1", len(prs))
	}
	pr := prs[0]
	if pr.ID != 4821 || pr.SourceBranch != "feature/retry" || pr.TargetBranch != "main" || !pr.IsDraft || !pr.HasConflicts() {
		t.Fatalf("got %+v", pr)
	}
	if pr.RepositoryID != "r1" || pr.ProjectID != "p1" || pr.Author.DisplayName != "Marta Ruiz" || pr.CreatedDate.IsZero() {
		t.Fatalf("got %+v", pr)
	}
	if len(pr.Reviewers) != 2 || pr.Reviewers[0].Vote != models.VoteApproved || !pr.Reviewers[0].Required || !pr.Reviewers[1].Group {
		t.Fatalf("got reviewers %+v", pr.Reviewers)
	}
	if pr.WebURL != "https://dev.azure.com/contoso/web/_git/phoenix%20api/pullrequest/4821" {
		t.Fatalf("got web url %q", pr.WebURL)
	}
	if !strings.Contains(seen[0].url, "searchCriteria.status=active") || !strings.Contains(seen[0].url, "api-version=7.1") {
		t.Fatalf("asked %s", seen[0].url)
	}
}

func TestPullRequestsCanBeScopedToOneRepository(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/git/repositories": `{"value":[{"id":"r2","name":"web"},{"id":"r9","name":"old","isDisabled":true},{"id":"r1","name":"API"}]}`}, &seen)
	repos, err := c.ListRepositories()
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 || repos[0].Name != "API" || repos[1].ID != "r2" {
		t.Fatalf("got %+v, want enabled repositories sorted by name", repos)
	}
	if _, err := c.ListPullRequests(models.PullRequestCompleted, "r1", models.PullRequestCriteria{}); err != nil {
		t.Fatal(err)
	}
	if got := seen[1].url; !strings.Contains(got, "/web/_apis/git/repositories/r1/pullrequests?") || !strings.Contains(got, "searchCriteria.status=completed") {
		t.Fatalf("asked %s", got)
	}
	criteria := models.PullRequestCriteria{CreatorID: "u1", ReviewerID: "u2", SourceRef: "refs/heads/feature/x", TargetRef: "refs/heads/main"}
	if _, err := c.ListPullRequests(models.PullRequestActive, "r1", criteria); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"searchCriteria.creatorId=u1", "searchCriteria.reviewerId=u2", "searchCriteria.sourceRefName=refs%2Fheads%2Ffeature%2Fx", "searchCriteria.targetRefName=refs%2Fheads%2Fmain", "%24top=200"} {
		if !strings.Contains(seen[2].url, want) {
			t.Errorf("search asked %s, missing %s", seen[2].url, want)
		}
	}
}

func TestFavoritesAreReadForTheWholeOrganization(t *testing.T) {
	var seen []recorded
	body := `{"value":[{"artifactId":"R1","artifactType":"Microsoft.TeamFoundation.Git.Repository"},{"artifactId":"gone","artifactIsDeleted":true},{"artifactId":"7"}]}`
	ids, err := clientRouting(t, map[string]string{"/favorite/favorites": body}, &seen).GetFavoriteArtifactIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "r1" || ids[1] != "7" {
		t.Fatalf("got %v", ids)
	}
	if seen[0].url != "https://dev.azure.com/contoso/_apis/favorite/favorites?api-version=7.1-preview" {
		t.Fatalf("asked %s", seen[0].url)
	}
}

func TestThreadsKeepAnchorsAndDropDeletedOnes(t *testing.T) {
	var seen []recorded
	body := `{"value":[` +
		`{"id":1,"status":"active","publishedDate":"2026-11-14T09:00:00Z","threadContext":{"filePath":"/a.go","rightFileStart":{"line":48,"offset":1}},` +
		`"comments":[{"id":1,"content":"Seed the jitter?","commentType":"text","author":{"id":"u1","displayName":"Deepak"},"usersLiked":[{"id":"u2","uniqueName":"marta@example.com"}]},` +
		`{"id":2,"parentCommentId":1,"content":"gone","isDeleted":true}]},` +
		`{"id":2,"publishedDate":"2026-11-14T10:00:00Z","comments":[{"id":1,"content":"Deepak voted 10","commentType":"system"}]},` +
		`{"id":3,"isDeleted":true,"comments":[{"id":1,"content":"x","commentType":"text"}]},` +
		`{"id":4,"status":"fixed","publishedDate":"2026-11-13T10:00:00Z","threadContext":{"filePath":"/b.go","leftFileStart":{"line":7,"offset":1}},` +
		`"comments":[{"id":1,"content":"old line","commentType":"text"}]}]}`
	threads, err := clientRouting(t, map[string]string{"/threads": body}, &seen).GetPullRequestThreads("r1", 4821, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 3 {
		t.Fatalf("got %d threads, want 3", len(threads))
	}
	if !threads[0].System() || threads[0].Resolvable() {
		t.Fatalf("newest thread should be the system one: %+v", threads[0])
	}
	if threads[1].FilePath != "/a.go" || threads[1].Line != 48 || threads[1].MultiLine() || len(threads[1].Comments) != 1 || !threads[1].Active() {
		t.Fatalf("got %+v", threads[1])
	}
	if c := threads[1].Comments[0]; len(c.Likes) != 1 || !c.LikedBy(models.Identity{UniqueName: "Marta@example.com"}) {
		t.Fatalf("the comment's likes must be kept, got %+v", c.Likes)
	}
	if !threads[2].LeftSide || threads[2].Line != 7 || threads[2].Active() {
		t.Fatalf("got %+v", threads[2])
	}
	if !strings.Contains(seen[0].url, "/threads?$iteration=3&api-version=7.1") {
		t.Fatalf("threads must be tracked to the shown iteration, asked %s", seen[0].url)
	}
	if models.ActiveThreads(threads) != 1 {
		t.Fatalf("got %d active threads, want 1", models.ActiveThreads(threads))
	}
}

func TestReviewWritesUseTheDocumentedVerbsAndBodies(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, nil, &seen)
	if err := c.VotePullRequest("r1", 7, "me-id", models.VoteWaitingForAuthor); err != nil {
		t.Fatal(err)
	}
	if err := c.CreatePullRequestThread("r1", 7, "Looks off", &models.ThreadAnchor{FilePath: "/a.go", Line: 12}); err != nil {
		t.Fatal(err)
	}
	if err := c.CreatePullRequestThread("r1", 7, "Whole loop", &models.ThreadAnchor{FilePath: "/a.go", Line: 12, EndLine: 15, EndOffset: 9}); err != nil {
		t.Fatal(err)
	}
	if err := c.ReplyToPullRequestThread("r1", 7, 3, 1, "Fixed"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetPullRequestThreadStatus("r1", 7, 3, models.ThreadFixed); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdatePullRequestComment("r1", 7, 3, 2, "Reworded"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeletePullRequestComment("r1", 7, 3, 2); err != nil {
		t.Fatal(err)
	}
	if err := c.LikePullRequestComment("r1", 7, 3, 1, true); err != nil {
		t.Fatal(err)
	}
	if err := c.LikePullRequestComment("r1", 7, 3, 1, false); err != nil {
		t.Fatal(err)
	}
	want := []struct{ method, path, body string }{
		{"PUT", "/git/repositories/r1/pullRequests/7/reviewers/me-id", `{"vote":-5}`},
		{"POST", "/git/repositories/r1/pullRequests/7/threads", `"rightFileStart":{"line":12,"offset":1}`},
		{"POST", "/git/repositories/r1/pullRequests/7/threads", `"rightFileStart":{"line":12,"offset":1},"rightFileEnd":{"line":15,"offset":9}`},
		{"POST", "/git/repositories/r1/pullRequests/7/threads/3/comments", `"parentCommentId":1`},
		{"PATCH", "/git/repositories/r1/pullRequests/7/threads/3", `{"status":"fixed"}`},
		{"PATCH", "/git/repositories/r1/pullRequests/7/threads/3/comments/2", `{"content":"Reworded"}`},
		{"DELETE", "/git/repositories/r1/pullRequests/7/threads/3/comments/2", ""},
		{"POST", "/git/repositories/r1/pullRequests/7/threads/3/comments/1/likes", ""},
		{"DELETE", "/git/repositories/r1/pullRequests/7/threads/3/comments/1/likes", ""},
	}
	if len(seen) != len(want) {
		t.Fatalf("made %d requests, want %d", len(seen), len(want))
	}
	for i, w := range want {
		if seen[i].method != w.method || !strings.Contains(seen[i].url, w.path+"?api-version=7.1") || !strings.Contains(seen[i].body, w.body) {
			t.Errorf("request %d: got %s %s %s, want %s %s containing %s", i, seen[i].method, seen[i].url, seen[i].body, w.method, w.path, w.body)
		}
	}
}

func TestChecksMergePoliciesWithCustomStatuses(t *testing.T) {
	var seen []recorded
	policies := `{"value":[` +
		`{"status":"approved","configuration":{"isBlocking":false,"isEnabled":true,"type":{"displayName":"Status"},"settings":{"statusName":"scan","statusGenre":"security"}}},` +
		`{"status":"running","configuration":{"isBlocking":true,"isEnabled":true,"type":{"displayName":"Build"},"settings":{"displayName":"phoenix-ci"}}},` +
		`{"status":"queued","configuration":{"isBlocking":true,"isEnabled":true,"type":{"displayName":"Minimum number of reviewers"},"settings":{"minimumApproverCount":2}}},` +
		`{"status":"notApplicable","configuration":{"isBlocking":true,"isEnabled":true,"type":{"displayName":"Required reviewers"}}},` +
		`{"status":"approved","configuration":{"isBlocking":true,"isEnabled":false,"type":{"displayName":"Disabled"}}}]}`
	statuses := `{"value":[` +
		`{"id":1,"state":"failed","description":"old run","context":{"name":"quality-gate","genre":"sonarqube"}},` +
		`{"id":2,"state":"succeeded","description":"gate passed","context":{"name":"quality-gate","genre":"sonarqube"}},` +
		`{"id":3,"state":"succeeded","context":{"name":"scan","genre":"security"}}]}`
	c := clientRouting(t, map[string]string{"/policy/evaluations": policies, "/statuses": statuses}, &seen)
	checks, err := c.GetPullRequestChecks(models.PullRequest{ID: 7, ProjectID: "p1", RepositoryID: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 4 {
		t.Fatalf("got %d checks, want 4: %+v", len(checks), checks)
	}
	if checks[0].Name != "Build: phoenix-ci" || checks[0].State != models.CheckRunning || !checks[0].Required {
		t.Fatalf("got %+v", checks[0])
	}
	if checks[1].Detail != "2 required" || checks[1].State != models.CheckPending {
		t.Fatalf("got %+v", checks[1])
	}
	if checks[2].Name != "security/scan" || checks[2].Required {
		t.Fatalf("got %+v", checks[2])
	}
	if checks[3].Name != "sonarqube/quality-gate" || checks[3].State != models.CheckPassed || checks[3].Detail != "gate passed" {
		t.Fatalf("the newest status per context must win: %+v", checks[3])
	}
	if models.BlockingChecks(checks) != 2 {
		t.Fatalf("got %d blocking checks, want 2", models.BlockingChecks(checks))
	}
	if !strings.Contains(seen[0].url, "artifactId=vstfs%3A%2F%2F%2FCodeReview%2FCodeReviewId%2Fp1%2F7") {
		t.Fatalf("asked %s", seen[0].url)
	}
}

func TestChangesUseTheLatestIterationAndSkipFolders(t *testing.T) {
	var seen []recorded
	iterations := `{"value":[{"id":1,"sourceRefCommit":{"commitId":"old"},"commonRefCommit":{"commitId":"base"}},` +
		`{"id":2,"sourceRefCommit":{"commitId":"head"},"commonRefCommit":{"commitId":"base"}}]}`
	changes := `{"changeEntries":[` +
		`{"changeType":"edit","item":{"path":"/src/b.go"}},` +
		`{"changeType":"add","item":{"path":"/src","isFolder":true}},` +
		`{"changeType":"add","item":{"path":"/src/a.go"}},` +
		`{"changeType":"delete","originalPath":"/old.go","item":{}},` +
		`{"changeType":"edit, rename","originalPath":"/docs/x.md","item":{"path":"/docs/y.md"}}]}`
	c := clientRouting(t, map[string]string{"/iterations/2/changes": changes, "/iterations": iterations}, &seen)
	got, err := c.GetPullRequestChanges("r1", 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Iteration != 2 || got.HeadCommit != "head" || got.BaseCommit != "base" {
		t.Fatalf("got %+v", got)
	}
	want := []models.PullRequestFile{
		{Path: "/docs/y.md", OriginalPath: "/docs/x.md", Kind: models.ChangeRename},
		{Path: "/old.go", OriginalPath: "/old.go", Kind: models.ChangeDelete},
		{Path: "/src/a.go", Kind: models.ChangeAdd},
		{Path: "/src/b.go", Kind: models.ChangeEdit},
	}
	if len(got.Files) != len(want) {
		t.Fatalf("got %+v", got.Files)
	}
	for i := range want {
		if got.Files[i] != want[i] {
			t.Errorf("file %d: got %+v, want %+v", i, got.Files[i], want[i])
		}
	}
}

func TestChecksKeepTheBuildTheyPointTo(t *testing.T) {
	var seen []recorded
	evaluations := `{"value":[{"status":"rejected","configuration":{"isEnabled":true,"isBlocking":true,"type":{"displayName":"Build"},"settings":{"displayName":"storefront-web-ci"}},"context":{"buildId":812}},` +
		`{"status":"approved","configuration":{"isEnabled":true,"isBlocking":true,"type":{"displayName":"Status"},"settings":{"statusName":"e2e","statusGenre":"ci"}}}]}`
	statuses := `{"value":[{"id":1,"state":"succeeded","context":{"name":"e2e","genre":"ci"},"targetUrl":"https://dev.azure.com/acme/web/_build/results?buildId=900&view=results"},` +
		`{"id":2,"state":"succeeded","context":{"name":"sonar","genre":"quality"},"targetUrl":"https://sonar.example.com/dashboard?id=web"}]}`
	c := clientRouting(t, map[string]string{"/policy/evaluations": evaluations, "/statuses": statuses}, &seen)
	checks, err := c.GetPullRequestChecks(models.PullRequest{ID: 4821, RepositoryID: "r1", ProjectID: "p1"})
	if err != nil {
		t.Fatal(err)
	}
	builds := models.BuildChecks(checks)
	if len(builds) != 2 || builds[0].BuildID != 812 || builds[1].BuildID != 900 || builds[1].Name != "ci/e2e" {
		t.Fatalf("got %+v", builds)
	}
}
