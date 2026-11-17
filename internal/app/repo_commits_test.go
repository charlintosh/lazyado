package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func sampleCommits() []models.Commit {
	commit := func(id, comment, author string, ago time.Duration, parents ...string) models.Commit {
		return models.Commit{CommitRef: models.CommitRef{ID: id, Comment: comment, Author: author, Date: time.Now().Add(-ago)}, Parents: parents}
	}
	return []models.Commit{
		commit("a3f9c21aa11", "Fix rounding in cart totals\n\nTotals drifted by a cent on large carts.", "Priya Nair", 48*time.Hour),
		commit("7c1e0b4bb22", "Merged PR 4812: Regional tax tables", "Tomás Ruiz", 12*24*time.Hour, "p1", "p2"),
		commit("ada9476cc33", "Apply coupon before tax", "Tomás Ruiz", 13*24*time.Hour),
	}
}

func openCommitsHub(t testing.TB, width, height int) App {
	t.Helper()
	a := openFilesHub(t, width, height)
	a = feed(t, a, press("g"), press("H"))
	if !a.inCommits() || !a.commits.loading || !a.commits.criteria.IsZero() {
		t.Fatalf("H on the repository root must open every commit, got %+v", a.commits.criteria)
	}
	return feed(t, a, commitsLoadedMsg{seq: a.commits.seq, commits: sampleCommits()})
}

func TestCommitsHubListsCommitsWithAPreview(t *testing.T) {
	a := openCommitsHub(t, 160, 45)
	if a.commits.detailFor != "a3f9c21aa11" {
		t.Fatalf("the first commit must be previewed, got %q", a.commits.detailFor)
	}
	a = feed(t, a, commitDetailMsg{id: "a3f9c21aa11", detail: models.CommitDetail{
		Commit: sampleCommits()[0],
		Files:  []models.CommitFile{{Path: "/src/cart.go", Kind: models.ChangeEdit}, {Path: "/src/money.go", Kind: models.ChangeAdd}},
	}})
	out := ansi.Strip(a.render())
	for _, want := range []string{"Repos / Commits / storefront-api", "Branch: main", "COMMITS", "Commits 3", "Message", "a3f9c21", "Priya Nair",
		"Merge  Merged PR 4812", "Totals drifted by a cent", "Changes · 2 files", "M src/cart.go", "A src/money.go"} {
		if !strings.Contains(out, want) {
			t.Errorf("Commits hub is missing %q:\n%s", want, out)
		}
	}
}

func TestFilesHistoryOpensTheCommitsOfThatPath(t *testing.T) {
	a := openFilesHub(t, 160, 45)
	a = feed(t, a, press("j"), press("H"))
	if !a.inCommits() || a.commits.criteria.Path != "/src" {
		t.Fatalf("H in Files must show the history of the selected path, got %+v", a.commits.criteria)
	}
	if a.commits.loadedFor.criteria.Path != "/src" {
		t.Fatal("the history must be asked of Azure DevOps by path")
	}
	a = feed(t, a, commitsLoadedMsg{seq: a.commits.seq, commits: sampleCommits()[:1]})
	if out := ansi.Strip(a.render()); !strings.Contains(out, "/path:/src") {
		t.Errorf("the path filter must be visible:\n%s", out)
	}
	a = feed(t, a, special(tea.KeyEscape))
	if !a.commits.criteria.IsZero() || a.commits.loadedFor.criteria.Path != "" {
		t.Fatal("esc must clear the path and reload every commit")
	}
}

func TestCommitSearchFiltersTextLocallyAndQualifiersOnTheServer(t *testing.T) {
	a := openCommitsHub(t, 160, 45)
	a = feed(t, a, press("/"), tea.PasteMsg{Content: "coupon"})
	if c := a.commits.panel.Selected(); c == nil || c.ID != "ada9476cc33" || a.commits.panel.Title() != "Commits 1 of 3" {
		t.Fatalf("text must filter the loaded commits as you type, got %+v %q", c, a.commits.panel.Title())
	}
	seq := a.commits.seq
	a, msgs := step(t, a, special(tea.KeyEnter))
	search, ok := find[panels.SearchCommitsMsg](msgs)
	if !ok {
		t.Fatalf("enter must search, got %v", msgs)
	}
	if a = feed(t, a, search); a.commits.seq != seq {
		t.Fatal("plain text must not reload from Azure DevOps")
	}
	a = feed(t, a, panels.SearchCommitsMsg{Query: "author:Tomás path:src/tax"})
	if a.commits.loadedFor.criteria != (models.CommitCriteria{Author: "Tomás", Path: "/src/tax"}) || a.commits.seq == seq {
		t.Fatalf("qualifiers must be sent to Azure DevOps, got %+v", a.commits.loadedFor.criteria)
	}
}

func TestCommitsLoadMoreAtTheEndAndDropStalePages(t *testing.T) {
	a := openCommitsHub(t, 160, 45)
	a = feed(t, a, commitsLoadedMsg{seq: a.commits.seq - 1, commits: sampleCommits()})
	if a.commits.panel.Count() != 3 {
		t.Fatal("a page from an earlier load must be dropped")
	}
	a.commits.panel.SetCommits(a.commits.commits, true)
	a, msgs := step(t, a, press("G"))
	more, ok := find[panels.MoreCommitsMsg](msgs)
	if !ok {
		t.Fatalf("reaching the end must ask for more, got %v", msgs)
	}
	a = feed(t, a, more, commitsLoadedMsg{seq: a.commits.seq, commits: []models.Commit{{CommitRef: models.CommitRef{ID: "0a1f3c7", Comment: "Initial skeleton"}}}})
	if a.commits.panel.Count() != 4 {
		t.Fatalf("the next page must be appended, got %d", a.commits.panel.Count())
	}
}

func TestEnterOnACommitBrowsesItsFiles(t *testing.T) {
	a := openCommitsHub(t, 160, 45)
	a = feed(t, a, special(tea.KeyEnter))
	if !a.inFiles() || a.code.version != (models.GitVersion{Kind: models.VersionCommit, Name: "a3f9c21aa11"}) {
		t.Fatalf("enter must browse the files at that commit, got %+v", a.code.version)
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Commit: a3f9c21") {
		t.Errorf("the header must show the commit:\n%s", out)
	}
	if !strings.HasSuffix(a.commitWebURL("a3f9c21aa11"), "/_git/storefront-api/commit/a3f9c21aa11") {
		t.Errorf("got %s", a.commitWebURL("a3f9c21aa11"))
	}
}

func TestCommitsHubFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {160, 45}, {120, 32}, {90, 26}, {70, 22}, {50, 14}} {
		a := openCommitsHub(t, sz[0], sz[1])
		assertFits(t, a.render(), sz[0], sz[1])
		a = feed(t, a, press("/"))
		assertFits(t, a.render(), sz[0], sz[1])
	}
}
