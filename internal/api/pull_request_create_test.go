package api

import (
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/models"
)

func TestBranchesAndDefaultBranchDropTheRefPrefix(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/repositories/r1/refs": `{"value":[{"name":"refs/heads/main"},{"name":"refs/heads/feature/1043-guest"}]}`,
		"/repositories/r1":      `{"id":"r1","defaultBranch":"refs/heads/main"}`,
	}, &seen)
	branches, err := c.ListBranches("r1")
	if err != nil || len(branches) != 2 || branches[0] != "feature/1043-guest" || branches[1] != "main" {
		t.Fatalf("got %v, %v", branches, err)
	}
	if !strings.Contains(seen[0].url, "/refs?filter=heads/") {
		t.Fatalf("branches must be read from refs/heads, asked %s", seen[0].url)
	}
	if def, err := c.DefaultBranch("r1"); err != nil || def != "main" {
		t.Fatalf("got %q, %v", def, err)
	}
}

func TestCompareBranchesCountsCommitsFilesAndReadsTheFirstCommits(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/diffs/commits": `{"aheadCount":1,"behindCount":4,"changeCounts":{"Edit":5,"Add":2}}`,
		"/commits":       `{"value":[{"comment":"Checkout: allow guests"}]}`,
	}, &seen)
	cmp, err := c.CompareBranches("r1", "feature/x", "main")
	if err != nil {
		t.Fatal(err)
	}
	if cmp.Ahead != 1 || cmp.Behind != 4 || cmp.Files != 7 || len(cmp.Commits) != 1 {
		t.Fatalf("got %+v", cmp)
	}
	for _, want := range []string{"baseVersion=main", "targetVersion=feature%2Fx", "baseVersionType=branch"} {
		if !strings.Contains(seen[0].url, want) {
			t.Errorf("compare asked %s, missing %s", seen[0].url, want)
		}
	}
	if !strings.Contains(seen[1].url, "searchCriteria.itemVersion.version=feature%2Fx") || !strings.Contains(seen[1].url, "searchCriteria.compareVersion.version=main") {
		t.Errorf("commits asked %s", seen[1].url)
	}
}

func TestCreatePullRequestSendsTheDocumentedBody(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/repositories/r1/pullrequests": `{"pullRequestId":4830,"title":"Guest checkout","status":"active","repository":{"id":"r1","name":"api"}}`,
	}, &seen)
	pr, err := c.CreatePullRequest("r1", models.NewPullRequest{
		Source: "feature/x", Target: "main", Title: "Guest checkout", Description: "Why", Draft: true,
		Reviewers: []models.NewReviewer{{ID: "u1", Required: true}, {ID: "u2"}}, WorkItems: []int{1043}, Labels: []string{"checkout"},
	})
	if err != nil || pr.ID != 4830 {
		t.Fatalf("got %+v, %v", pr, err)
	}
	r := seen[0]
	if r.method != "POST" || !strings.Contains(r.url, "/git/repositories/r1/pullrequests?api-version=7.1") {
		t.Fatalf("got %s %s", r.method, r.url)
	}
	for _, want := range []string{
		`"sourceRefName":"refs/heads/feature/x"`, `"targetRefName":"refs/heads/main"`, `"isDraft":true`,
		`"reviewers":[{"id":"u1","isRequired":true},{"id":"u2","isRequired":false}]`, `"workItemRefs":[{"id":"1043"}]`, `"labels":[{"name":"checkout"}]`,
	} {
		if !strings.Contains(r.body, want) {
			t.Errorf("body %s is missing %s", r.body, want)
		}
	}
}

func TestPullRequestTemplatePrefersTheBranchSpecificOne(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/items": "## What changed\n"}, &seen)
	if got := c.PullRequestTemplate("r1", "main"); got != "## What changed\n" {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(seen[0].url, "pull_request_template%2Fbranches%2Fmain.md") || !strings.Contains(seen[0].url, "versionDescriptor.version=main") {
		t.Fatalf("the branch-specific template must be tried first on the target branch, asked %s", seen[0].url)
	}
}
