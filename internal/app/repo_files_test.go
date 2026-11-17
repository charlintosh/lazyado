package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charmbracelet/x/ansi"
)

func sampleRepos() []models.Repository {
	return []models.Repository{
		{ID: "r1", Name: "storefront-api", DefaultBranch: "main",
			RemoteURL: "https://dev.azure.com/acme/storefront/_git/storefront-api",
			SSHURL:    "git@ssh.example.com:v3/acme/storefront/storefront-api",
			WebURL:    "https://dev.azure.com/acme/storefront/_git/storefront-api"},
		{ID: "r2", Name: "web-app", DefaultBranch: "develop"},
	}
}

func lastChange(id, comment string, ago time.Duration) models.CommitRef {
	return models.CommitRef{ID: id, Comment: comment, Author: "Priya Nair", Date: time.Now().Add(-ago)}
}

func rootItems() []models.GitItem {
	return []models.GitItem{
		{Path: "/src", Folder: true, Last: lastChange("7c1e0b4aa", "Split tax rules by region", 12*24*time.Hour)},
		{Path: "/README.md", Last: lastChange("e91d3aa00", "Document the checkout service", 21*24*time.Hour)},
		{Path: "/go.mod", Last: lastChange("a3f9c21bb", "Fix rounding in cart totals", 48*time.Hour)},
	}
}

func openFilesHub(t testing.TB, width, height int) App {
	t.Helper()
	a := newTestApp(t, width, height)
	a = feed(t, a, press("E"))
	if !a.inFiles() || !a.pr.reposLoading {
		t.Fatal("E must open the Files hub and load the repositories")
	}
	a = feed(t, a, repositoriesLoadedMsg{repos: sampleRepos()})
	a = feed(t, a, modals.RepositoryChosenMsg{Repository: models.Repository{ID: "r1", Name: "storefront-api"}})
	a = feed(t, a, refsLoadedMsg{seq: a.code.refsSeq, branches: []string{"feature/x", "main"}, tags: []string{"v1.0"}})
	return feed(t, a, folderLoadedMsg{seq: a.code.seq, path: "/", items: rootItems()})
}

func TestFilesHubAsksForASingleRepositoryWhenThereAreSeveral(t *testing.T) {
	a := newTestApp(t, 160, 45)
	a = feed(t, a, press("E"), repositoriesLoadedMsg{repos: sampleRepos()})
	picker := topModal[modals.RepositoryPickerModal](t, a)
	if out := ansi.Strip(picker.View()); strings.Contains(out, "All repositories") || !strings.Contains(out, "web-app") {
		t.Fatalf("the Files hub needs one repository, so the picker must not offer All repositories:\n%s", out)
	}
	a = feed(t, a, modals.RepositoryChosenMsg{Repository: models.Repository{ID: "r1", Name: "storefront-api"}})
	if a.modals.open() || a.code.repo.RemoteURL == "" || a.code.version != models.BranchVersion("main") {
		t.Fatalf("choosing must close the picker, keep the clone URLs and start on the default branch, got %+v %+v", a.code.repo, a.code.version)
	}
	if a.pr.repo.ID != "" {
		t.Error("choosing a repository for Files must not change the pull request scope")
	}
	if a.savedState().CodeRepository != "storefront-api" {
		t.Error("the Files repository must be remembered between runs")
	}
}

func TestFilesHubRestoresTheSavedRepositoryWithoutAsking(t *testing.T) {
	a := newTestApp(t, 160, 45)
	a.code.savedRepo = "web-app"
	a = feed(t, a, press("E"), repositoriesLoadedMsg{repos: sampleRepos()})
	if a.modals.open() || a.code.repo.ID != "r2" || a.code.version.Name != "develop" {
		t.Fatalf("got repo %+v version %+v", a.code.repo, a.code.version)
	}
	if !a.code.files.requested[folderKey+"/"] {
		t.Error("the root folder must load as soon as the repository is known")
	}
}

func TestFilesHubShowsTheTreeTheFolderTableAndItsReadme(t *testing.T) {
	a := openFilesHub(t, 160, 45)
	if !a.code.files.requested[fileKey+"/README.md"] {
		t.Fatal("a folder with a README.md must load it for the content pane")
	}
	a = feed(t, a, repoFileLoadedMsg{seq: a.code.seq, path: "/README.md", text: "# Storefront API\n\nCart, pricing and tax."})
	out := ansi.Strip(a.render())
	for _, want := range []string{"Repos / Files / storefront-api", "Repo: storefront-api", "Branch: main", "Clone", "FILES",
		"src/", "go.mod", "storefront-api · 3 items", "Last change", "7c1e0b4", "Split tax rules by region", "Cart, pricing and tax."} {
		if !strings.Contains(out, want) {
			t.Errorf("Files hub is missing %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "src/") > strings.Index(out, "go.mod") {
		t.Error("folders must be listed before files")
	}
}

func TestFilesHubExpandsAFolderAndShowsAFileWithLineNumbers(t *testing.T) {
	a := openFilesHub(t, 160, 45)
	a = feed(t, a, press("j"), press("l"))
	if !a.code.files.requested[folderKey+"/src"] {
		t.Fatal("expanding a folder must load its children")
	}
	a = feed(t, a, folderLoadedMsg{seq: a.code.seq, path: "/src", items: []models.GitItem{{Path: "/src/cart.go", Last: lastChange("a3f9c21bb", "Fix rounding", time.Hour)}}})
	a = feed(t, a, press("j"))
	if sel := a.code.files.tree.Selected(); sel == nil || sel.Path != "/src/cart.go" {
		t.Fatalf("got %+v", sel)
	}
	if a.code.files.requested[fileKey+"/src/cart.go"] {
		t.Fatal("moving onto a file must wait for the debounce before loading it")
	}
	a = feed(t, a, repoFileDueMsg{seq: a.code.seq, path: "/src/cart.go"})
	if !a.code.files.requested[fileKey+"/src/cart.go"] {
		t.Fatal("the file must load once the cursor rests on it")
	}
	a = feed(t, a, repoFileLoadedMsg{seq: a.code.seq, path: "/src/cart.go", text: "package cart\n\nfunc Total() int {\n\treturn 0\n}\n"})
	out := ansi.Strip(a.render())
	for _, want := range []string{"storefront-api / src / cart.go", "1  package cart", "3  func Total() int {", "4      return 0"} {
		if !strings.Contains(out, want) {
			t.Errorf("file view is missing %q:\n%s", want, out)
		}
	}
	a, msgs := step(t, a, special(tea.KeyEnter))
	open, ok := find[panels.OpenRepoFileMsg](msgs)
	if !ok || open.Item.Path != "/src/cart.go" {
		t.Fatalf("enter on a file must open it, got %v", msgs)
	}
	if a = feed(t, a, open); a.code.files.pane != paneContent {
		t.Fatal("enter on a file must focus its contents")
	}
	if a = feed(t, a, press("j"), press("j")); a.code.files.content.Line() != 3 {
		t.Fatalf("j must move the line cursor in the contents, got line %d", a.code.files.content.Line())
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "┃    3  func Total() int {") {
		t.Errorf("the cursor line must carry the focus bar like the pull request diff:\n%s", out)
	}
	a = feed(t, a, special(tea.KeyEscape))
	if a.code.files.pane != paneTree {
		t.Fatal("esc must go back to the tree")
	}
}

func TestFilesHubDropsResponsesFromAnotherBranch(t *testing.T) {
	a := openFilesHub(t, 160, 45)
	old := a.code.seq
	a = feed(t, a, press("b"))
	topModal[modals.ChoiceModal](t, a)
	a = feed(t, a, modals.VersionChosenMsg{Version: models.GitVersion{Kind: models.VersionTag, Name: "v1.0"}})
	if a.code.seq == old || a.code.version.Name != "v1.0" {
		t.Fatalf("choosing a tag must reload the tree, got %+v", a.code.version)
	}
	a = feed(t, a, folderLoadedMsg{seq: old, path: "/", items: rootItems()})
	if a.code.files.tree.Loaded("/") {
		t.Fatal("a folder loaded for the previous branch must be dropped")
	}
	if out := ansi.Strip(a.render()); !strings.Contains(out, "Tag: v1.0") {
		t.Errorf("the header must show the tag:\n%s", out)
	}
}

func TestCloneDialogCopiesTheHTTPSAndSSHURLs(t *testing.T) {
	a := openFilesHub(t, 160, 45)
	a = feed(t, a, press("c"))
	out := ansi.Strip(topModal[modals.ChoiceModal](t, a).View())
	for _, want := range []string{"Clone storefront-api", "HTTPS", "SSH", "git@ssh.example.com:v3/acme/storefront/storefront-api", "git clone https://"} {
		if !strings.Contains(out, want) {
			t.Errorf("clone dialog is missing %q:\n%s", want, out)
		}
	}
	_, msgs := step(t, a, special(tea.KeyEnter))
	copied, ok := find[modals.CopyRequestMsg](msgs)
	if !ok || copied.Text != sampleRepos()[0].RemoteURL {
		t.Fatalf("enter must copy the HTTPS URL, got %+v", msgs)
	}
	a = feed(t, a, special(tea.KeyDown))
	_, msgs = step(t, a, special(tea.KeyEnter))
	if copied, _ := find[modals.CopyRequestMsg](msgs); copied.Text != sampleRepos()[0].SSHURL {
		t.Fatalf("the second option must copy the SSH URL, got %+v", msgs)
	}
}

func TestFilesWebURLPointsAtThePathAndVersion(t *testing.T) {
	a := openFilesHub(t, 160, 45)
	got := a.fileWebURL("/src/cart.go", share.Lines{})
	if !strings.Contains(got, "path=%2Fsrc%2Fcart.go") || !strings.Contains(got, "version=GBmain") || strings.Contains(got, "line=") {
		t.Fatalf("got %s", got)
	}
	if got := a.fileWebURL("/src/cart.go", share.Lines{From: 12, To: 12}); !strings.Contains(got, "line=12") || !strings.Contains(got, "lineEnd=13") {
		t.Fatalf("the browser must open at the cursor line, got %s", got)
	}
}

func TestFilesHubFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{200, 50}, {160, 45}, {120, 32}, {90, 26}, {70, 22}, {50, 14}} {
		a := openFilesHub(t, sz[0], sz[1])
		assertFits(t, a.render(), sz[0], sz[1])
		a = feed(t, a, press("c"))
		assertFits(t, a.render(), sz[0], sz[1])
		a = feed(t, a, special(tea.KeyEscape), press("b"))
		assertFits(t, a.render(), sz[0], sz[1])
	}
}
