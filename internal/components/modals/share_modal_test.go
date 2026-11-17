package modals

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charmbracelet/x/ansi"
)

const shareURL = "https://dev.azure.com/contoso/Storefront/_workitems/edit/1043"

func shareTarget() share.Target {
	return share.WorkItem(models.WorkItem{ID: 1043, Title: "Guest checkout", Type: models.WorkItemTypePBI, WebURL: shareURL})
}

func copied(t *testing.T, msg tea.Msg) CopyRequestMsg {
	t.Helper()
	c, ok := msg.(CopyRequestMsg)
	if !ok {
		t.Fatalf("got %#v, want a copy request", msg)
	}
	return c
}

func TestShareModalListsEveryFormatWithItsLetter(t *testing.T) {
	out := ansi.Strip(NewShareModal(testEnv(), shareTarget()).View())
	for _, want := range []string{"Share", "Product Backlog Item 1043 · Guest checkout", "u  Link", "i  ID", "h  Mention", "g  GitHub commit link", "m  Markdown link", "t  Title and link", "#1043", "AB#1043"} {
		if !strings.Contains(out, want) {
			t.Errorf("share menu is missing %q:\n%s", want, out)
		}
	}
}

func TestShareModalLettersCopyTheirFormat(t *testing.T) {
	cases := map[rune]string{
		'u': shareURL,
		'i': "1043",
		'h': "#1043",
		'g': "AB#1043",
		'm': "[1043 Guest checkout](" + shareURL + ")",
		'Y': shareURL,
	}
	for letter, want := range cases {
		_, msg := keyPress(NewShareModal(testEnv(), shareTarget()), letter, 0)
		if c := copied(t, msg); c.Text != want {
			t.Errorf("%c copied %q, want %q", letter, c.Text, want)
		}
	}
}

func TestShareModalLetterForAMissingFormatDoesNothing(t *testing.T) {
	_, msg := keyPress(NewShareModal(testEnv(), shareTarget()), 'p', 0)
	if msg != nil {
		t.Fatalf("a work item has no permalink, got %#v", msg)
	}
}

func TestShareModalEnterCopiesTheHighlightedFormat(t *testing.T) {
	var m Modal = NewShareModal(testEnv(), shareTarget())
	m, _ = keyPress(m, tea.KeyDown, 0)
	_, msg := keyPress(m, tea.KeyEnter, 0)
	c := copied(t, msg)
	if c.Text != "1043" || c.Notice != "ID copied for work item 1043" {
		t.Fatalf("got %+v", c)
	}
}

func TestShareModalOpensTheLinkInTheBrowser(t *testing.T) {
	_, msg := keyPress(NewShareModal(testEnv(), shareTarget()), 'o', 0)
	open, ok := msg.(OpenURLRequestMsg)
	if !ok || open.URL != shareURL {
		t.Fatalf("got %#v", msg)
	}
}

func TestShareModalPreviewsMultiLineSnippets(t *testing.T) {
	code := share.Code{RepoURL: "https://dev.azure.com/contoso/Storefront/_git/api", RepoName: "api", Path: "/a.go", Version: models.BranchVersion("main")}
	target := share.FileLines(code, share.Lines{From: 1, To: 2, Code: []string{"package a", "var x = 1"}})
	out := ansi.Strip(NewShareModal(testEnv(), target).View())
	if !strings.Contains(out, "```go … (5 lines)") {
		t.Fatalf("the snippet must preview its first line and size:\n%s", out)
	}
}
