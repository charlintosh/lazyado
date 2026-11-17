package panels

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"

	"github.com/charmbracelet/x/ansi"
)

func richComment() models.Comment {
	return models.Comment{
		ID:          1,
		CreatedBy:   "Marco Lima",
		CreatedDate: time.Now().Add(-2 * time.Hour),
		Text: `<div><a href="#" data-vss-mention="version:2.0,abc">@Dana Reyes</a> can you check this?</div>` +
			`<ul><li>first <b>bold</b></li><li><i>second</i></li></ul><pre><code>go test ./...</code></pre>`,
	}
}

func TestCommentsRenderRichMarkdown(t *testing.T) {
	out, offsets := RenderComments([]models.Comment{richComment()}, CommentListConfig{Width: 60, Selected: -1})
	plain := ansi.Strip(out)
	for _, want := range []string{"@Dana Reyes can you check this?", "• first bold", "• second", "go test ./..."} {
		if !strings.Contains(plain, want) {
			t.Errorf("rendered comment is missing %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "<") || strings.Contains(plain, "data-vss-mention") {
		t.Errorf("raw HTML leaked into the comment:\n%s", plain)
	}
	if len(offsets) != 1 || offsets[0] != 0 {
		t.Errorf("offsets = %v, want [0]", offsets)
	}
}

func TestCommentThreadEditCarriesWorkItem(t *testing.T) {
	me := models.Identity{ID: "a1b2"}
	mine := []models.Comment{{ID: 1, CreatedByID: "a1b2"}, {ID: 2, CreatedByID: "A1B2"}}
	thread := NewCommentThread(keys.DefaultKeyMap()).WithIdentity(me).WithComments(42, mine)
	thread, _ = thread.Update(press("j"))
	_, cmd := thread.Update(press("e"))
	msg, ok := cmd().(EditCommentMsg)
	if !ok || msg.WorkItemID != 42 || msg.Comment.ID != 2 {
		t.Fatalf("got %#v, want edit of comment 2 on work item 42", cmd())
	}
	if again := thread.WithComments(42, []models.Comment{{ID: 1}}); again.Cursor() != 0 {
		t.Fatalf("cursor %d must clamp to the shorter list", again.Cursor())
	}
	if other := thread.WithComments(7, []models.Comment{{ID: 1}, {ID: 2}}); other.Cursor() != 0 {
		t.Fatalf("cursor must reset for another work item, got %d", other.Cursor())
	}
}

func numberedComments(n int) []models.Comment {
	out := make([]models.Comment, n)
	for i := range out {
		out[i] = models.Comment{ID: i + 1, CreatedBy: "Dana Reyes", Text: fmt.Sprintf("<div>comment-%02d</div>", i)}
	}
	return out
}

func TestDetailViewScrollsToSelectedComment(t *testing.T) {
	item := &models.WorkItem{ID: 5, Title: "Item", Type: models.WorkItemTypePBI, Description: "Body"}
	d := NewDetailView(styles.DefaultStyles(), keys.DefaultKeyMap())
	d.SetSize(100, 20)
	d.SetItem(item)
	d.SetComments(5, numberedComments(10))
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !d.commentFocused {
		t.Fatal("tab must focus the discussion")
	}
	for range 9 {
		d, _ = d.Update(press("j"))
	}
	if view := ansi.Strip(d.View()); !strings.Contains(view, "comment-09") || strings.Contains(view, "comment-00") {
		t.Fatalf("expected the view scrolled to the last comment:\n%s", view)
	}
	for range 9 {
		d, _ = d.Update(press("k"))
	}
	if view := ansi.Strip(d.View()); !strings.Contains(view, "comment-00") {
		t.Fatalf("first comment is not visible after navigating back:\n%s", view)
	}
}

func TestCommentsPanelScrollsToSelectedComment(t *testing.T) {
	c := NewCommentsPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	c.SetSize(60, 12)
	c.SetFocused(true)
	c.StartLoading(5)
	c.SetComments(5, numberedComments(10))
	for range 9 {
		c, _ = c.Update(press("j"))
	}
	if view := ansi.Strip(c.View()); !strings.Contains(view, "comment-09") || strings.Contains(view, "comment-00") {
		t.Fatalf("expected the panel scrolled to the last comment:\n%s", view)
	}
}

func TestCommentThreadOnlyModifiesOwnComments(t *testing.T) {
	me := models.Identity{ID: "me", UniqueName: "dana@contoso.com"}
	comments := []models.Comment{
		{ID: 1, CreatedBy: "Dana Kim", CreatedByID: "me"},
		{ID: 2, CreatedBy: "Dana Kim", CreatedByID: "someone-else"},
	}
	thread := NewCommentThread(keys.DefaultKeyMap()).WithIdentity(me).WithComments(42, comments)
	for _, k := range []string{"e", "d"} {
		_, cmd := thread.Update(press(k))
		switch cmd().(type) {
		case EditCommentMsg, DeleteCommentMsg:
		default:
			t.Fatalf("%s on my own comment produced %#v", k, cmd())
		}
	}
	thread, _ = thread.Update(press("j"))
	if thread.CanModify() {
		t.Fatal("a namesake's comment must not be editable")
	}
	for _, k := range []string{"e", "d"} {
		_, cmd := thread.Update(press(k))
		if msg, ok := cmd().(CommentReadOnlyMsg); !ok || !msg.IdentityKnown {
			t.Fatalf("%s on someone else's comment produced %#v", k, cmd())
		}
	}
	for _, b := range thread.KeyBindings() {
		if b.Help().Desc == "edit comment" || b.Help().Desc == "delete comment" {
			t.Fatalf("hints offer %q on someone else's comment", b.Help().Desc)
		}
	}
	_, cmd := NewCommentThread(keys.DefaultKeyMap()).WithComments(42, comments).Update(press("e"))
	if msg, ok := cmd().(CommentReadOnlyMsg); !ok || msg.IdentityKnown {
		t.Fatalf("without an identity got %#v", cmd())
	}
}

func TestWorkItemCommentsUseTheSharedCard(t *testing.T) {
	me := models.Identity{ID: "me"}
	now := time.Now()
	comments := []models.Comment{
		{ID: 1, Text: "mine", CreatedBy: "Dana Kim", CreatedByID: "me", CreatedDate: now, ModifiedDate: now.Add(time.Minute)},
		{ID: 2, Text: "theirs", CreatedBy: "Ben Okafor", CreatedByID: "ben", CreatedDate: now, Likes: 2, LikedByMe: true},
	}
	for _, width := range []int{80, 44} {
		out, offsets := RenderComments(comments, CommentListConfig{Width: width, Selected: 1, Me: me, Hints: "+ unlike"})
		plain := ansi.Strip(out)
		for _, want := range []string{"╭─ DK Dana Kim (you) · just now · edited", "╭─ BO Ben Okafor · just now", "♥ 2 · you ─╮", "╰─ + unlike"} {
			if !strings.Contains(plain, want) {
				t.Errorf("at %d columns the discussion is missing %q:\n%s", width, want, plain)
			}
		}
		if strings.Contains(plain, "Ben Okafor (you)") {
			t.Errorf("at %d columns someone else's comment is marked as yours", width)
		}
		lines := strings.Split(plain, "\n")
		if offsets[1] >= len(lines) || !strings.Contains(lines[offsets[1]], "Ben Okafor") {
			t.Errorf("at %d columns offset %d does not start the second card", width, offsets[1])
		}
	}
	narrow := ansi.Strip(renderCommentsAt(comments, minBoxedWidth-1))
	if strings.Contains(narrow, "╭") || !strings.Contains(narrow, "│ BO Ben Okafor") {
		t.Errorf("narrower than %d columns the discussion must use the rail:\n%s", minBoxedWidth, narrow)
	}
}

func renderCommentsAt(comments []models.Comment, width int) string {
	out, _ := RenderComments(comments, CommentListConfig{Width: width, Selected: -1})
	return out
}

func TestCommentThreadLikesOnlyOtherPeoplesComments(t *testing.T) {
	me := models.Identity{ID: "me"}
	comments := []models.Comment{
		{ID: 1, CreatedBy: "Dana Kim", CreatedByID: "me"},
		{ID: 2, CreatedBy: "Ben Okafor", CreatedByID: "ben"},
		{ID: 3, CreatedBy: "Ben Okafor", CreatedByID: "ben", Likes: 1, LikedByMe: true},
	}
	thread := NewCommentThread(keys.DefaultKeyMap()).WithIdentity(me).WithComments(42, comments)
	if _, cmd := thread.Update(press("+")); cmd != nil {
		t.Fatalf("liking my own comment produced %#v", cmd())
	}
	thread, _ = thread.Update(press("j"))
	_, cmd := thread.Update(press("+"))
	if msg, ok := cmd().(LikeCommentMsg); !ok || !msg.Like || msg.WorkItemID != 42 || msg.Comment.ID != 2 {
		t.Fatalf("got %#v, want a like of comment 2 on work item 42", cmd())
	}
	if out := ansi.Strip(mustRender(thread)); !strings.Contains(out, "╰─ + like") {
		t.Fatalf("the focused comment must offer a like:\n%s", out)
	}
	thread, _ = thread.Update(press("j"))
	_, cmd = thread.Update(press("+"))
	if msg, ok := cmd().(LikeCommentMsg); !ok || msg.Like {
		t.Fatalf("got %#v, want the like removed", cmd())
	}
	stranger := NewCommentThread(keys.DefaultKeyMap()).WithComments(42, comments)
	stranger, _ = stranger.Update(press("j"))
	if _, cmd := stranger.Update(press("+")); cmd != nil {
		t.Fatal("without an identity nothing can be liked")
	}
}

func mustRender(thread CommentThread) string {
	out, _ := thread.Render(80, true)
	return out
}
