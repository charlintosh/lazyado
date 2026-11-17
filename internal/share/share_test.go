package share

import (
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/models"
)

const web = "https://dev.azure.com/contoso/Storefront"

func texts(t Target) map[Kind]string {
	out := map[Kind]string{}
	for _, o := range t.Options {
		out[o.Kind] = o.Text
	}
	return out
}

func TestWorkItemOffersLinkIDMentionsAndMarkdown(t *testing.T) {
	item := models.WorkItem{ID: 1043, Title: "Guest checkout [beta]", Type: models.WorkItemTypePBI, WebURL: web + "/_workitems/edit/1043"}
	target := WorkItem(item)
	got := texts(target)
	want := map[Kind]string{
		Link:     web + "/_workitems/edit/1043",
		ID:       "1043",
		Mention:  "#1043",
		GitHub:   "AB#1043",
		Markdown: `[1043 Guest checkout \[beta\]](` + web + "/_workitems/edit/1043)",
		Title:    "Product Backlog Item 1043: Guest checkout [beta] — " + web + "/_workitems/edit/1043",
	}
	for kind, text := range want {
		if got[kind] != text {
			t.Errorf("kind %d: got %q, want %q", kind, got[kind], text)
		}
	}
	if o, _ := target.Default(); o.Kind != Link {
		t.Errorf("the first option must be the link, got %+v", o)
	}
	if n := target.Notice(target.Options[0]); n != "Link copied for work item 1043" {
		t.Errorf("got notice %q", n)
	}
}

func TestOptionsWithoutTextAreLeftOut(t *testing.T) {
	target := PullRequest(models.PullRequest{ID: 7, Title: "Draft"})
	if _, ok := target.Find(Link); ok {
		t.Error("a pull request without a web URL must not offer a link")
	}
	if _, ok := target.Default(); !ok {
		t.Error("the ID must still be offered")
	}
	if o, _ := target.Default(); o.Text != "!7" {
		t.Errorf("got %+v", o)
	}
}

func TestFileLinesUseTheBranchAndAPermalinkAtTheCommit(t *testing.T) {
	code := Code{RepoURL: web + "/_git/api", RepoName: "api", Path: "/src/cart.go", Version: models.BranchVersion("main"), Commit: "a3f9c21bb0"}
	lines := Lines{From: 3, To: 4, Code: []string{"func Total() int {", "\treturn 0"}}
	target := FileLines(code, lines)
	got := texts(target)
	for _, part := range []string{"path=%2Fsrc%2Fcart.go", "version=GBmain", "line=3", "lineEnd=5", "lineStartColumn=1", "lineEndColumn=1", "_a=contents"} {
		if !strings.Contains(got[Link], part) {
			t.Errorf("link %q is missing %q", got[Link], part)
		}
	}
	if !strings.Contains(got[Permalink], "version=GCa3f9c21bb0") || !strings.Contains(got[Permalink], "line=3") {
		t.Errorf("got permalink %q", got[Permalink])
	}
	if got[Path] != "src/cart.go:3-4" {
		t.Errorf("got path %q", got[Path])
	}
	wantSnippet := "```go\nfunc Total() int {\n\treturn 0\n```\n" + got[Permalink]
	if got[Snippet] != wantSnippet {
		t.Errorf("got snippet %q, want %q", got[Snippet], wantSnippet)
	}
	if !strings.HasPrefix(got[Markdown], "[cart.go#L3-4](") {
		t.Errorf("got markdown %q", got[Markdown])
	}
	if target.Subject != "api · src/cart.go · lines 3–4 on main" {
		t.Errorf("got subject %q", target.Subject)
	}
}

func TestSingleLineAndMissingCommit(t *testing.T) {
	code := Code{RepoURL: web + "/_git/api", RepoName: "api", Path: "/go.mod", Version: models.BranchVersion("main")}
	got := texts(FileLines(code, Lines{From: 1, To: 1, Code: []string{"module api"}}))
	if _, ok := got[Permalink]; ok {
		t.Error("without a commit there is no permalink")
	}
	if got[Path] != "go.mod:1" {
		t.Errorf("got %q", got[Path])
	}
	if !strings.HasSuffix(got[Snippet], got[Link]) {
		t.Errorf("without a permalink the snippet must link to the branch, got %q", got[Snippet])
	}
}

func TestSnippetFenceOutgrowsBackticksInTheCode(t *testing.T) {
	snippet := Lines{From: 1, To: 1, Code: []string{"s := ```x```"}}.Snippet("/a.md", "")
	if !strings.HasPrefix(snippet, "````markdown\n") || !strings.HasSuffix(snippet, "\n````") {
		t.Fatalf("got %q", snippet)
	}
}

func TestPullRequestLinesPinTheCommitOfTheirSide(t *testing.T) {
	pr := models.PullRequest{ID: 4826, Title: "Retries", WebURL: web + "/_git/api/pullrequest/4826"}
	got := texts(PullRequestLines(pr, "head123", "/src/client.go", Lines{From: 5, To: 7, Code: []string{"a", "b", "c"}}))
	if got[Link] != web+"/_git/api/pullrequest/4826?_a=files&path=%2Fsrc%2Fclient.go" {
		t.Errorf("got link %q", got[Link])
	}
	if !strings.HasPrefix(got[Permalink], web+"/_git/api?") || !strings.Contains(got[Permalink], "version=GChead123") || !strings.Contains(got[Permalink], "lineEnd=8") {
		t.Errorf("got permalink %q", got[Permalink])
	}
	if !strings.Contains(got[Markdown], "client.go#L5-7 in !4826") {
		t.Errorf("got markdown %q", got[Markdown])
	}
}

func TestRunOffersNumberIDLogsAndPipeline(t *testing.T) {
	run := models.PipelineRun{ID: 98765, Number: "20261117.3", PipelineID: 42, PipelineName: "api CI", Result: "failed", WebURL: web + "/_build/results?buildId=98765"}
	logs := run.WebURL + "&view=logs&j=job1"
	got := texts(Run(run, logs, "Integration tests"))
	want := map[Kind]string{
		ID:           "20261117.3",
		RunID:        "98765",
		Logs:         logs,
		PipelineLink: web + "/_build?definitionId=42",
		Title:        "api CI 20261117.3 (failed) — " + run.WebURL,
	}
	for kind, text := range want {
		if got[kind] != text {
			t.Errorf("kind %d: got %q, want %q", kind, got[kind], text)
		}
	}
	if _, ok := texts(Run(run, run.WebURL, ""))[Logs]; ok {
		t.Error("without a selected job there is no logs link")
	}
}

func TestReleaseLinksItsPipelineAndSummarisesStages(t *testing.T) {
	r := models.Release{ID: 311, Name: "Release-311", DefinitionID: 7, Definition: "Billing API", WebURL: web + "/_release?releaseId=311&_a=release-summary",
		Stages: []models.ReleaseStage{{Name: "Dev", State: models.StageSucceeded}, {Name: "Prod", State: models.StageDeploying}}}
	got := texts(Release(r))
	if got[PipelineLink] != web+"/_release?definitionId=7" {
		t.Errorf("got %q", got[PipelineLink])
	}
	if !strings.HasPrefix(got[Title], "Billing API Release-311 (Dev: deployed, Prod: deploying) — ") {
		t.Errorf("got %q", got[Title])
	}
}
