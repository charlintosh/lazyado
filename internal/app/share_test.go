package ui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func openShare(t *testing.T, a App) (App, string) {
	t.Helper()
	a, msgs := step(t, a, press("y"))
	a = feed(t, a, msgs...)
	menu := topModal[modals.ShareModal](t, a)
	return a, ansi.Strip(menu.View())
}

func shareAs(t *testing.T, a App, letter string) modals.CopyRequestMsg {
	t.Helper()
	_, msgs := step(t, a, press(letter))
	c, ok := find[modals.CopyRequestMsg](msgs)
	if !ok {
		t.Fatalf("%s must copy a format, got %v", letter, msgs)
	}
	return c
}

func TestShareKeyOpensTheMenuInEveryHub(t *testing.T) {
	cases := map[string]struct {
		open func(testing.TB, int, int) App
		want string
	}{
		"pull requests":     {openPullRequestHub, "Pull request !"},
		"review overview":   {openReview, "Pull request !4821"},
		"review files":      {openFiles, "!4821 · internal/webhook/client.go"},
		"builds":            {openPipelinesHub, "storefront-web-ci"},
		"run":               {openRunView, "20261117.6"},
		"releases":          {openReleasesHub, "smartHotel"},
		"files":             {openFilesHub, "storefront-api · / on main"},
		"commits":           {openCommitsHub, "Commit "},
		"branches":          {openBranchesHub, "Branch "},
		"tags":              {openTagsHub, "Tag "},
		"work items (list)": {func(t testing.TB, w, h int) App { return newTestApp(t, w, h) }, "Share"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, out := openShare(t, c.open(t, 160, 45))
			if !strings.Contains(out, c.want) || !strings.Contains(out, "copy link") {
				t.Fatalf("the share menu must name %q:\n%s", c.want, out)
			}
		})
	}
}

func TestShareWorkItemCopiesItsFormats(t *testing.T) {
	a := openDetailOf(t, 1043)
	a, out := openShare(t, a)
	if !strings.Contains(out, "1043") {
		t.Fatalf("got:\n%s", out)
	}
	if c := shareAs(t, a, "h"); c.Text != "#1043" {
		t.Errorf("h must copy the mention, got %+v", c)
	}
	if c := shareAs(t, a, "g"); c.Text != "AB#1043" {
		t.Errorf("g must copy the GitHub commit link, got %+v", c)
	}
}

func TestShareLinkNowSkipsTheMenu(t *testing.T) {
	a := openPullRequestHub(t, 160, 45)
	pr := *a.pr.panel.Selected()
	a, msgs := step(t, a, press("Y"))
	req, ok := find[panels.ShareMsg](msgs)
	if !ok || !req.Now || req.Target.Subject != "Pull request !"+strconv.Itoa(pr.ID)+" · "+pr.Title {
		t.Fatalf("Y must ask to copy the link right away, got %+v", msgs)
	}
	if a.modals.open() {
		t.Fatal("Y must not open the menu")
	}
}

func TestShareSelectedLinesInAPullRequest(t *testing.T) {
	a := openFiles(t, 160, 45)
	a = feed(t, a, press("j"), press("j"), press("j"), press("v"), press("j"), press("j"))
	if status := ansi.Strip(a.render()); !strings.Contains(status, "share 3 lines") {
		t.Fatalf("the status bar must offer to share the selection:\n%s", status[len(status)-200:])
	}
	a, out := openShare(t, a)
	if !strings.Contains(out, "lines 4–6") {
		t.Fatalf("the menu must name the lines:\n%s", out)
	}
	if c := shareAs(t, a, "f"); c.Text != "internal/webhook/client.go:4-6" {
		t.Errorf("got %q", c.Text)
	}
	permalink := shareAs(t, a, "p").Text
	for _, part := range []string{"version=GChead", "line=4", "lineEnd=7", "path=%2Finternal%2Fwebhook%2Fclient.go"} {
		if !strings.Contains(permalink, part) {
			t.Errorf("permalink %q is missing %q", permalink, part)
		}
	}
	if snippet := shareAs(t, a, "c").Text; !strings.HasPrefix(snippet, "```go\n\tvar resp *http.Response\n") {
		t.Errorf("got snippet %q", snippet)
	}
}

func openCartFile(t *testing.T) App {
	t.Helper()
	a := openFilesHub(t, 160, 45)
	a = feed(t, a, press("j"), press("l"))
	a = feed(t, a, folderLoadedMsg{seq: a.code.seq, path: "/src", items: []models.GitItem{{Path: "/src/cart.go", Last: lastChange("a3f9c21bb", "Fix rounding", time.Hour)}}})
	a = feed(t, a, press("j"), repoFileDueMsg{seq: a.code.seq, path: "/src/cart.go"})
	a = feed(t, a, repoFileLoadedMsg{seq: a.code.seq, path: "/src/cart.go", text: "package cart\n\nfunc Total() int {\n\treturn 0\n}\n"})
	return feed(t, a, panels.OpenRepoFileMsg{Item: *a.code.files.tree.Selected()})
}

func TestShareSelectedLinesInAFile(t *testing.T) {
	a := openCartFile(t)
	a = feed(t, a, press("v"), press("j"), press("j"))
	if status := ansi.Strip(a.render()); !strings.Contains(status, "share 3 lines") {
		t.Fatalf("v must select lines in the Files hub:\n%s", status[len(status)-200:])
	}
	a, out := openShare(t, a)
	if !strings.Contains(out, "lines 1–3 on main") {
		t.Fatalf("got:\n%s", out)
	}
	if c := shareAs(t, a, "f"); c.Text != "src/cart.go:1-3" {
		t.Errorf("got %q", c.Text)
	}
	if c := shareAs(t, a, "c"); c.Text != "```go\npackage cart\n\nfunc Total() int {\n```\n"+shareAs(t, a, "p").Text {
		t.Errorf("got snippet %q", c.Text)
	}
	if c := shareAs(t, a, "p"); !strings.Contains(c.Text, "version=GCa3f9c21bb") || !strings.Contains(c.Text, "lineEnd=4") {
		t.Errorf("the permalink must pin the file's last commit, got %q", c.Text)
	}
}

func TestMarkdownFilesHaveAFileTabToSelectAndShareLines(t *testing.T) {
	a := openFilesHub(t, 160, 45)
	for i := 0; i < 5 && a.code.files.tree.Selected().Path != "/README.md"; i++ {
		a = feed(t, a, press("j"))
	}
	a = feed(t, a, repoFileDueMsg{seq: a.code.seq, path: "/README.md"})
	a = feed(t, a, repoFileLoadedMsg{seq: a.code.seq, path: "/README.md", text: "# Storefront API\n\nCart, pricing and tax.\n"})
	a = feed(t, a, panels.OpenRepoFileMsg{Item: *a.code.files.tree.Selected()})
	out := ansi.Strip(a.render())
	if !strings.Contains(out, "README.md  File  Preview") || strings.Contains(out, "1  # Storefront API") {
		t.Fatalf("a Markdown file must open on its rendered Preview, with File and Preview tabs:\n%s", out)
	}
	a = feed(t, a, press("1"))
	if out := ansi.Strip(a.render()); !strings.Contains(out, "1  # Storefront API") || !strings.Contains(out, "file/preview") {
		t.Fatalf("1 must show the highlighted source with line numbers:\n%s", out)
	}
	a = feed(t, a, press("v"), press("j"), press("j"))
	a, _ = openShare(t, a)
	if c := shareAs(t, a, "f"); c.Text != "README.md:1-3" {
		t.Errorf("got %q", c.Text)
	}
	a, msgs := step(t, a, special(tea.KeyEscape))
	a = feed(t, a, append(msgs, press("2"))...)
	if out := ansi.Strip(a.render()); strings.Contains(out, "1  # Storefront API") || a.code.files.content.Selecting() {
		t.Fatalf("2 must go back to the Preview and drop the selection:\n%s", out)
	}
	a = feed(t, a, press("]"))
	if !a.code.files.content.MarkdownSource() {
		t.Fatal("] must toggle back to the File tab")
	}
}

func TestEscClearsTheFileSelectionBeforeLeaving(t *testing.T) {
	a := openCartFile(t)
	a = feed(t, a, press("v"), press("j"), special(tea.KeyEscape))
	if a.code.files.pane != paneContent || a.code.files.content.Selecting() {
		t.Fatal("esc must first clear the selection and stay in the contents")
	}
	if a = feed(t, a, special(tea.KeyEscape)); a.code.files.pane != paneTree {
		t.Fatal("a second esc must go back to the tree")
	}
}
