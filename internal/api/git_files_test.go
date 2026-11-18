package api

import (
	"errors"
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/models"
)

func TestListRepositoriesKeepsCloneURLsAndDefaultBranch(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/git/repositories": `{"value":[{"id":"r1","name":"storefront-api","defaultBranch":"refs/heads/main",` +
			`"remoteUrl":"https://dev.azure.com/contoso/web/_git/storefront-api",` +
			`"sshUrl":"git@ssh.example.com:v3/contoso/web/storefront-api","webUrl":"https://dev.azure.com/contoso/web/_git/storefront-api"}]}`,
	}, &seen)
	repos, err := c.ListRepositories()
	if err != nil || len(repos) != 1 {
		t.Fatalf("got %v, %v", repos, err)
	}
	r := repos[0]
	if r.DefaultBranch != "main" || !strings.HasPrefix(r.RemoteURL, "https://") || !strings.HasPrefix(r.SSHURL, "git@ssh.") || r.WebURL == "" {
		t.Fatalf("got %+v", r)
	}
}

func TestListItemsReadsOneLevelWithLastChangeAndSkipsTheFolderItself(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/items": `{"value":[` +
			`{"path":"/src","isFolder":true,"gitObjectType":"tree"},` +
			`{"path":"/src/cart.go","gitObjectType":"blob","latestProcessedChange":{"commitId":"a3f9c21aa","comment":"Fix rounding\n\nBody","author":{"name":"Priya Nair","date":"2026-11-15T15:03:00Z"}}},` +
			`{"path":"/src/tax","isFolder":true,"gitObjectType":"tree"},` +
			`{"path":"/src/vendor-lib","gitObjectType":"commit"}]}`,
	}, &seen)
	items, err := c.ListItems("r1", "/src", models.BranchVersion("feature/x"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || !items[0].Folder || items[0].Name() != "tax" || items[1].Name() != "cart.go" {
		t.Fatalf("folders must come first and the folder itself and submodules must be dropped, got %+v", items)
	}
	if last := items[1].Last; last.ID != "a3f9c21aa" || last.Subject() != "Fix rounding" || last.Author != "Priya Nair" || last.Date.IsZero() {
		t.Fatalf("got %+v", last)
	}
	for _, want := range []string{"scopePath=%2Fsrc", "recursionLevel=OneLevel", "latestProcessedChange=true", "versionDescriptor.versionType=branch", "versionDescriptor.version=feature%2Fx"} {
		if !strings.Contains(seen[0].url, want) {
			t.Errorf("asked %s, missing %s", seen[0].url, want)
		}
	}
}

func TestGetItemTextAsksForTheVersionAndRefusesLargeFiles(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/items": "package cart\n"}, &seen)
	text, err := c.GetItemText("r1", "/src/cart.go", models.GitVersion{Kind: models.VersionTag, Name: "v1.0"})
	if err != nil || text != "package cart\n" {
		t.Fatalf("got %q, %v", text, err)
	}
	if !strings.Contains(seen[0].url, "versionDescriptor.versionType=tag") || !strings.Contains(seen[0].url, "%24format=text") {
		t.Errorf("asked %s", seen[0].url)
	}
	c.maxFileBytes = 4
	if _, err := c.GetItemText("r1", "/src/cart.go", models.BranchVersion("main")); !errors.Is(err, models.ErrFileTooLarge) {
		t.Fatalf("a file over the limit must fail with ErrFileTooLarge, got %v", err)
	}
}

func TestListTagNamesDropsTheRefPrefix(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{"/refs": `{"value":[{"name":"refs/tags/v2.0"},{"name":"refs/tags/v1.0"}]}`}, &seen)
	tags, err := c.ListTagNames("r1")
	if err != nil || len(tags) != 2 || tags[0] != "v1.0" {
		t.Fatalf("got %v, %v", tags, err)
	}
	if !strings.Contains(seen[0].url, "filter=tags/") {
		t.Errorf("asked %s", seen[0].url)
	}
}

func TestListCommitsSendsTheVersionFiltersAndPage(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/commits": `{"value":[{"commitId":"a3f9c21aa","comment":"Fix rounding","author":{"name":"Priya Nair","date":"2026-11-15T15:03:00Z"},` +
			`"committer":{"name":"Priya Nair","date":"2026-11-15T15:04:00Z"},"changeCounts":{"Add":1,"Edit":2}}]}`,
	}, &seen)
	commits, err := c.ListCommits("r1", models.BranchVersion("main"), models.CommitCriteria{Author: "Priya", Path: "/src"}, 100)
	if err != nil || len(commits) != 1 {
		t.Fatalf("got %v, %v", commits, err)
	}
	if got := commits[0]; got.ID != "a3f9c21aa" || got.Adds != 1 || got.Edits != 2 || got.Committer != "Priya Nair" {
		t.Fatalf("got %+v", got)
	}
	for _, want := range []string{"searchCriteria.itemVersion.version=main", "searchCriteria.itemVersion.versionType=branch",
		"searchCriteria.author=Priya", "searchCriteria.itemPath=%2Fsrc", "searchCriteria.%24top=100", "searchCriteria.%24skip=100"} {
		if !strings.Contains(seen[0].url, want) {
			t.Errorf("asked %s, missing %s", seen[0].url, want)
		}
	}
}

func TestGetCommitDetailReadsParentsAndChangedFiles(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/commits/a3f9c21aa":         `{"commitId":"a3f9c21aa","comment":"Merged PR 4790: Cart refactor\n\nDetails","parents":["p1","p2"],"author":{"name":"Ana Ito"}}`,
		"/commits/a3f9c21aa/changes": `{"changes":[{"changeType":"edit","item":{"path":"/src/cart.go"}},{"changeType":"add","item":{"path":"/src/money","isFolder":true}},{"changeType":"delete, sourceRename","item":{"path":"/old.go"}}]}`,
	}, &seen)
	d, err := c.GetCommitDetail("r1", "a3f9c21aa")
	if err != nil {
		t.Fatal(err)
	}
	if !d.Commit.IsMerge() || d.Commit.Body() != "Details" || d.Commit.Subject() != "Merged PR 4790: Cart refactor" {
		t.Fatalf("got %+v", d.Commit)
	}
	if len(d.Files) != 2 || d.Files[0].Kind != models.ChangeEdit || d.Files[1].Kind != models.ChangeDelete {
		t.Fatalf("folders must be dropped and kinds mapped, got %+v", d.Files)
	}
}

func TestListAllItemsIndexesTheWholeTreeInOneRequestAndStopsAtTheCap(t *testing.T) {
	var seen []recorded
	c := clientRouting(t, map[string]string{
		"/items": `{"count":5,"value":[` +
			`{"path":"/","isFolder":true,"gitObjectType":"tree","commitId":"2b4b23cdd"},` +
			`{"path":"/src","isFolder":true,"gitObjectType":"tree","commitId":"2b4b23cdd"},` +
			`{"path":"/src/cart.go","gitObjectType":"blob","commitId":"2b4b23cdd"},` +
			`{"path":"/src/vendor-lib","gitObjectType":"commit"},` +
			`{"path":"/README.md","gitObjectType":"blob"}]}`,
	}, &seen)
	x, err := c.ListAllItems("r1", models.BranchVersion("main"), 100)
	if err != nil {
		t.Fatal(err)
	}
	if x.Len() != 2 || x.Commit != "2b4b23cdd" || x.Truncated || !x.Has("/src/cart.go") || len(x.Children("/src")) != 1 {
		t.Fatalf("got %d files, commit %q, truncated %v", x.Len(), x.Commit, x.Truncated)
	}
	if len(seen) != 1 || !strings.Contains(seen[0].url, "recursionLevel=Full") || strings.Contains(seen[0].url, "latestProcessedChange") {
		t.Fatalf("the index must be one plain request, got %+v", seen)
	}
	x, err = c.ListAllItems("r1", models.BranchVersion("main"), 2)
	if err != nil || !x.Truncated || x.Len() != 1 {
		t.Fatalf("the cap must stop reading: %v, truncated %v, %d files", err, x != nil && x.Truncated, x.Len())
	}
}
