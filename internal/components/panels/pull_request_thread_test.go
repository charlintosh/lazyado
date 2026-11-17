package panels

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func borderedThread() models.PullRequestThread {
	now := time.Now()
	return models.PullRequestThread{ID: 4, Status: models.ThreadActive, FilePath: "/a.go", Line: 3, Date: now,
		Comments: []models.PullRequestComment{
			{ID: 1, AuthorName: "Marco Lima", Content: "Seed the jitter?", Date: now, Likes: []models.Identity{{ID: "u2"}}},
			{ID: 2, ParentID: 1, AuthorID: "me", AuthorName: "Dana Reyes", Content: "Pushed a fix.", Date: now},
		}}
}

func TestThreadCommentsGetTheirOwnBorder(t *testing.T) {
	st := threadStyle{width: 80, selected: true, comment: 2, me: models.Identity{ID: "me"}, hints: "r reply  e edit"}
	r := renderThread(borderedThread(), st)
	lines := strings.Split(ansi.Strip(r.text), "\n")
	for _, want := range []string{"╭─ ML Marco Lima", "♥ 1 ─╮", "│ Seed the jitter?", "  ╭─ DR Dana Reyes (you)", "  ╰─ r reply  e edit ─"} {
		if !strings.Contains(ansi.Strip(r.text), want) {
			t.Errorf("thread is missing %q:\n%s", want, ansi.Strip(r.text))
		}
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w > 80 {
			t.Errorf("line %d is %d columns wide, over the 80 given: %q", i, w, l)
		}
	}
	if !strings.Contains(lines[r.top], "DR Dana Reyes") || !strings.HasPrefix(strings.TrimSpace(lines[r.bottom]), "╰─ r reply") {
		t.Fatalf("the focus span must cover the focused card, got %d-%d", r.top, r.bottom)
	}
}

func TestNarrowThreadsFallBackToTheRail(t *testing.T) {
	r := renderThread(borderedThread(), threadStyle{width: minBoxedWidth - 1})
	out := ansi.Strip(r.text)
	if strings.Contains(out, "╭") || !strings.Contains(out, "│ ML Marco Lima") {
		t.Fatalf("a thread narrower than %d columns must use the rail:\n%s", minBoxedWidth, out)
	}
}

func TestFoldedThreadIsOneBorderedRow(t *testing.T) {
	thread := borderedThread()
	thread.Status = models.ThreadFixed
	r := renderThread(thread, threadStyle{width: 80, selected: true, comment: 1, hints: "Enter expand"})
	lines := strings.Split(ansi.Strip(r.text), "\n")
	if len(lines) != 3 || !strings.Contains(lines[1], "RESOLVED") || !strings.Contains(lines[2], "╰─ Enter expand") {
		t.Fatalf("a folded thread must be one bordered row with its hints in the bottom edge:\n%s", strings.Join(lines, "\n"))
	}
}
